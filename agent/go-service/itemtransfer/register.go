package itemtransfer

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

func Register() {
	maa.AgentServerAddResourceSink(&resourcePathSink{})
	maa.AgentServerRegisterCustomAction(
		"ItemTransferFallbackAction",
		&ItemTransferFallbackAction{},
	)
	maa.AgentServerRegisterCustomAction(
		"ItemTransferOCRAction",
		&ItemTransferOCRAction{},
	)
}
