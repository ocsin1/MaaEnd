package seizedeliveryjobs

import maa "github.com/MaaXYZ/maa-framework-go/v4"

func Register() {
	maa.AgentServerRegisterCustomRecognition("SeizeDeliveryJobsScanTargetRecognition", &SeizeDeliveryJobsScanTargetRecognition{})
	maa.AgentServerRegisterCustomRecognition("SeizeDeliveryJobsTimedFallbackRecognition", &SeizeDeliveryJobsTimedFallbackRecognition{})
	maa.AgentServerRegisterCustomAction("SeizeDeliveryJobsScanTargetAction", &SeizeDeliveryJobsScanTargetAction{})
	maa.AgentServerRegisterCustomAction("SeizeDeliveryJobsResetScanStateAction", &SeizeDeliveryJobsResetScanStateAction{})
	maa.AgentServerRegisterCustomAction("SeizeDeliveryJobsTimedFallbackResetAction", &SeizeDeliveryJobsTimedFallbackResetAction{})
	maa.AgentServerRegisterCustomAction("SeizeDeliveryJobsDepartureAction", &SeizeDeliveryJobsDepartureAction{})
}
