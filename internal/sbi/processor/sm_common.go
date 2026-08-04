package processor

import (
	smf_context "github.com/free5gc/smf/internal/context"
)

func (p *Processor) RemoveSMContextFromAllNF(smContext *smf_context.SMContext, sendNotification bool) {
	smContext.SetState(smf_context.InActive)

	// Notify BSF about PCF binding release if we have a binding ID
	if smContext.BSFBindingID != "" {
		p.Consumer().NotifyBSFBindingRelease(smContext)
	}

	// remove SM Policy Association
	if smContext.SMPolicyID != "" {
		if err := p.Consumer().SendSMPolicyAssociationTermination(smContext); err != nil {
			smContext.Log.Errorf("SM Policy Termination failed: %s", err)
		} else {
			smContext.SMPolicyID = ""
		}
	}

	if smf_context.GetSelf().Ues.UeExists(smContext.Supi) {
		problemDetails, err := p.Consumer().UnSubscribe(smContext)
		if problemDetails != nil {
			smContext.Log.Errorf("SDM UnSubscription Failed Problem[%+v]", problemDetails)
		} else if err != nil {
			smContext.Log.Errorf("SDM UnSubscription Error[%+v]", err)
		}
	}

	// A PDU Session can fail after its serving-SMF registration has already
	// been created in UDM (for example, when PFCP establishment fails).  Do not
	// leave that registration behind when the common rollback path removes the
	// local SM context, otherwise UDM consumers can resolve a PDU Session that
	// no longer exists in this SMF.
	if smContext.UeCmRegistered {
		problemDetails, err := p.Consumer().UeCmDeregistration(smContext)
		if problemDetails != nil {
			if problemDetails.Cause != CONTEXT_NOT_FOUND {
				smContext.Log.Errorf("UECM_DeRegistration Failed Problem[%+v]", problemDetails)
			}
		} else if err != nil {
			smContext.Log.Errorf("UECM_DeRegistration Error[%+v]", err)
		} else {
			smContext.Log.Traceln("UECM_DeRegistration successful")
		}
	}

	// Because the amfUE who called this SMF API is being locked until the API Handler returns,
	// sending SMContext Status Notification should run asynchronously
	// so that this function returns immediately.
	go p.sendSMContextStatusNotificationAndRemoveSMContext(smContext, sendNotification)
}

func (p *Processor) sendSMContextStatusNotificationAndRemoveSMContext(
	smContext *smf_context.SMContext, sendNotification bool,
) {
	smContext.SMLock.Lock()
	defer smContext.SMLock.Unlock()

	if sendNotification && len(smContext.SmStatusNotifyUri) != 0 {
		p.SendReleaseNotification(smContext)
	}

	smf_context.RemoveSMContext(smContext.Ref)
}

func (p *Processor) SendReleaseNotification(smContext *smf_context.SMContext) {
	// Use go routine to send Notification to prevent blocking the handling process
	problemDetails, err := p.Consumer().SendSMContextStatusNotification(smContext.SmStatusNotifyUri)
	if problemDetails != nil || err != nil {
		if problemDetails != nil {
			smContext.Log.Warnf("Send SMContext Status Notification Problem[%+v]", problemDetails)
		}

		if err != nil {
			smContext.Log.Warnf("Send SMContext Status Notification Error[%v]", err)
		}
	} else {
		smContext.Log.Traceln("Send SMContext Status Notification successfully")
	}
}
