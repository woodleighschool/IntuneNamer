package intune

import (
	"context"
	"encoding/json"
	"fmt"

	azidentity "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	kiota "github.com/microsoft/kiota-abstractions-go"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	devicemanagement "github.com/microsoftgraph/msgraph-sdk-go/devicemanagement"
	msgraphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	odataerrors "github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	msgraphusers "github.com/microsoftgraph/msgraph-sdk-go/users"
)

// Credentials contains Azure AD authentication details.
type Credentials struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	GraphBaseURL string
}

// Client provides access to Microsoft Graph APIs.
type Client struct {
	graph   *msgraph.GraphServiceClient
	baseURL string
}

func NewClient(creds Credentials) (*Client, error) {
	if creds.TenantID == "" || creds.ClientID == "" || creds.ClientSecret == "" {
		return nil, fmt.Errorf("all credential fields are required")
	}

	baseURL := creds.GraphBaseURL
	if baseURL == "" {
		baseURL = "https://graph.microsoft.com/v1.0"
	}

	cred, err := azidentity.NewClientSecretCredential(creds.TenantID, creds.ClientID, creds.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("graph credential: %w", err)
	}

	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("graph client: %w", err)
	}

	return &Client{
		graph:   graphClient,
		baseURL: baseURL,
	}, nil
}

// ListManagedDevices fetches all managed devices.
func (c *Client) ListManagedDevices(ctx context.Context) ([]ManagedDevice, error) {
	if c.graph == nil {
		return nil, fmt.Errorf("graph client missing")
	}

	builder := c.graph.DeviceManagement().ManagedDevices()
	adapter := c.graph.GetAdapter()
	pageSize := int32(200)
	fields := selectFields()

	var devices []ManagedDevice

	for {
		resp, err := builder.Get(ctx, &devicemanagement.ManagedDevicesRequestBuilderGetRequestConfiguration{
			QueryParameters: &devicemanagement.ManagedDevicesRequestBuilderGetQueryParameters{
				Top:    &pageSize,
				Select: fields,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("list managed devices: %w", err)
		}

		for _, item := range resp.GetValue() {
			if item == nil {
				continue
			}
			devices = append(devices, convertManagedDevice(item))
		}

		next := resp.GetOdataNextLink()
		if next == nil || len(*next) == 0 {
			break
		}

		builder = devicemanagement.NewManagedDevicesRequestBuilder(*next, adapter)
	}

	return devices, nil
}

// SetDeviceName updates a device name using the Graph API.
func (c *Client) SetDeviceName(ctx context.Context, deviceID, desired string) error {
	if c.graph == nil {
		return fmt.Errorf("graph client missing")
	}

	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	if desired == "" {
		return fmt.Errorf("device name is required")
	}

	payload, err := json.Marshal(map[string]string{"deviceName": desired})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	// setDeviceName requires beta endpoint
	urlTemplate := "https://graph.microsoft.com/beta/deviceManagement/managedDevices/{managedDeviceId}/setDeviceName"
	pathParams := map[string]string{
		"managedDeviceId": deviceID,
	}

	req := kiota.NewRequestInformationWithMethodAndUrlTemplateAndPathParameters(
		kiota.POST,
		urlTemplate,
		pathParams,
	)
	req.SetStreamContentAndContentType(payload, "application/json")

	if err := c.graph.GetAdapter().SendNoContent(ctx, req, odataErrorMapping()); err != nil {
		return fmt.Errorf("set device name: %w", err)
	}

	return nil
}

// GetUser fetches an Entra user by ID or UPN.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user identifier is required")
	}

	fields := []string{
		"id",
		"displayName",
		"userPrincipalName",
		"mailNickname",
		"department",
	}

	resp, err := c.graph.Users().ByUserId(userID).Get(ctx, &msgraphusers.UserItemRequestBuilderGetRequestConfiguration{
		QueryParameters: &msgraphusers.UserItemRequestBuilderGetQueryParameters{
			Select: fields,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	user := &User{
		ID:                derefString(resp.GetId()),
		DisplayName:       derefString(resp.GetDisplayName()),
		UserPrincipalName: derefString(resp.GetUserPrincipalName()),
		MailNickname:      derefString(resp.GetMailNickname()),
		Department:        derefString(resp.GetDepartment()),
	}

	return user, nil
}

// GetUserGroups fetches all group memberships for a user.
func (c *Client) GetUserGroups(ctx context.Context, userID string) ([]Group, error) {
	if userID == "" {
		return nil, fmt.Errorf("user identifier is required")
	}

	builder := c.graph.Users().ByUserId(userID).TransitiveMemberOf().GraphGroup()
	adapter := c.graph.GetAdapter()

	fields := []string{
		"id",
		"displayName",
	}
	pageSize := int32(100)

	var groups []Group

	for {
		resp, err := builder.Get(ctx, &msgraphusers.ItemTransitiveMemberOfGraphGroupRequestBuilderGetRequestConfiguration{
			QueryParameters: &msgraphusers.ItemTransitiveMemberOfGraphGroupRequestBuilderGetQueryParameters{
				Select: fields,
				Top:    &pageSize,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("list user groups: %w", err)
		}

		for _, item := range resp.GetValue() {
			if item == nil {
				continue
			}

			groups = append(groups, Group{
				ID:          derefString(item.GetId()),
				DisplayName: derefString(item.GetDisplayName()),
			})
		}

		next := resp.GetOdataNextLink()
		if next == nil || len(*next) == 0 {
			break
		}

		builder = msgraphusers.NewItemTransitiveMemberOfGraphGroupRequestBuilder(*next, adapter)
	}

	return groups, nil
}

func convertManagedDevice(item msgraphmodels.ManagedDeviceable) ManagedDevice {
	device := ManagedDevice{
		ID:                    derefString(item.GetId()),
		DeviceName:            derefString(item.GetDeviceName()),
		OperatingSystem:       derefString(item.GetOperatingSystem()),
		OSVersion:             derefString(item.GetOsVersion()),
		SerialNumber:          derefString(item.GetSerialNumber()),
		UserID:                derefString(item.GetUserId()),
		UserDisplayName:       derefString(item.GetUserDisplayName()),
		UserPrincipalName:     derefString(item.GetUserPrincipalName()),
		AzureADDeviceID:       derefString(item.GetAzureADDeviceId()),
		DeviceCategoryDisplay: derefString(item.GetDeviceCategoryDisplayName()),
		EnrollmentProfileName: derefString(item.GetEnrollmentProfileName()),
		Manufacturer:          derefString(item.GetManufacturer()),
		Model:                 derefString(item.GetModel()),
		WiFiMacAddress:        derefString(item.GetWiFiMacAddress()),
		IMEI:                  derefString(item.GetImei()),
		EasDeviceID:           derefString(item.GetEasDeviceId()),
		PhoneNumber:           derefString(item.GetPhoneNumber()),
	}

	if agent := item.GetManagementAgent(); agent != nil {
		device.ManagementAgent = agent.String()
	}

	if state := item.GetManagementState(); state != nil {
		device.ManagementState = state.String()
	}

	if compliance := item.GetComplianceState(); compliance != nil {
		device.ComplianceState = compliance.String()
	}

	if enrolled := item.GetEnrolledDateTime(); enrolled != nil {
		device.EnrolledDateTime = *enrolled
	}

	if lastSync := item.GetLastSyncDateTime(); lastSync != nil {
		device.LastSyncDateTime = *lastSync
	}

	if registered := item.GetAzureADRegistered(); registered != nil {
		device.AzureADRegistered = *registered
	}

	return device
}

func odataErrorMapping() kiota.ErrorMappings {
	return kiota.ErrorMappings{
		"4XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
		"5XX": odataerrors.CreateODataErrorFromDiscriminatorValue,
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func selectFields() []string {
	return []string{
		"id",
		"deviceName",
		"operatingSystem",
		"osVersion",
		"serialNumber",
		"userId",
		"userDisplayName",
		"userPrincipalName",
		"managementAgent",
		"managementState",
		"azureADDeviceId",
		"complianceState",
		"deviceCategoryDisplayName",
		"enrollmentProfileName",
		"manufacturer",
		"model",
		"lastSyncDateTime",
		"enrolledDateTime",
		"wiFiMacAddress",
		"imei",
		"easDeviceId",
		"azureADRegistered",
		"phoneNumber",
	}
}
