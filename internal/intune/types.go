package intune

import "time"

// ManagedDevice represents an Intune device.
type ManagedDevice struct {
	ID                    string    `json:"id"`
	DeviceName            string    `json:"deviceName"`
	OperatingSystem       string    `json:"operatingSystem"`
	OSVersion             string    `json:"osVersion"`
	SerialNumber          string    `json:"serialNumber"`
	UserID                string    `json:"userId"`
	UserDisplayName       string    `json:"userDisplayName"`
	UserPrincipalName     string    `json:"userPrincipalName"`
	ManagementAgent       string    `json:"managementAgent"`
	ManagementState       string    `json:"managementState"`
	AzureADDeviceID       string    `json:"azureADDeviceId"`
	ComplianceState       string    `json:"complianceState"`
	DeviceCategoryDisplay string    `json:"deviceCategoryDisplayName"`
	EnrollmentProfileName string    `json:"enrollmentProfileName"`
	Manufacturer          string    `json:"manufacturer"`
	Model                 string    `json:"model"`
	LastSyncDateTime      time.Time `json:"lastSyncDateTime"`
	EnrolledDateTime      time.Time `json:"enrolledDateTime"`
	WiFiMacAddress        string    `json:"wiFiMacAddress"`
	IMEI                  string    `json:"imei"`
	EasDeviceID           string    `json:"easDeviceId"`
	AzureADRegistered     bool      `json:"azureADRegistered"`
	PhoneNumber           string    `json:"phoneNumber"`
}
