/*
 * SMF Nupf_EventExposure resolver helpers
 *
 * This file provides helper functions to resolve a SUPI into the UE IP and
 * selected UPF API root needed for Nupf_EventExposure subscriptions.
 */

package context

import (
	"errors"
	"net"
	"strings"
)

var (
	ErrNoActiveSession   = errors.New("no active session for supi")
	ErrMultipleSessions  = errors.New("multiple active sessions for supi")
	ErrNoUeIpAddress     = errors.New("ue ip address is not available")
	ErrNoSelectedUpf     = errors.New("selected upf is not available")
	ErrNoUpfEventApiRoot = errors.New("upf event exposure apiRoot is not configured")
)

// UpfEventExposureTarget holds resolved data required for Nupf_EventExposure requests.
type UpfEventExposureTarget struct {
	UeIpAddress net.IP
	SelectedUpf *UPNode
	UpfApiRoot  string
}

// GetSMContextBySupi returns the single SMContext matching the SUPI.
// If multiple contexts exist for the SUPI, ErrMultipleSessions is returned.
func GetSMContextBySupi(supi string) (*SMContext, error) {
	var selected *SMContext
	count := 0

	smContextPool.Range(func(_, value any) bool {
		smContext := value.(*SMContext)
		if smContext.Supi == supi {
			selected = smContext
			count++
			if count > 1 {
				return false
			}
		}
		return true
	})

	switch count {
	case 0:
		return nil, ErrNoActiveSession
	case 1:
		return selected, nil
	default:
		return nil, ErrMultipleSessions
	}
}

// ResolveUpfEventExposureTarget resolves the UE IP and UPF apiRoot for the SUPI.
func ResolveUpfEventExposureTarget(supi string) (*UpfEventExposureTarget, error) {
	smContext, err := GetSMContextBySupi(supi)
	if err != nil {
		return nil, err
	}

	if smContext.PDUAddress == nil {
		return nil, ErrNoUeIpAddress
	}
	if smContext.SelectedUPF == nil {
		return nil, ErrNoSelectedUpf
	}

	apiRoot := strings.TrimSpace(smContext.SelectedUPF.NupfEeApiRoot)
	if apiRoot == "" {
		return nil, ErrNoUpfEventApiRoot
	}

	return &UpfEventExposureTarget{
		UeIpAddress: smContext.PDUAddress,
		SelectedUpf: smContext.SelectedUPF,
		UpfApiRoot:  apiRoot,
	}, nil
}
