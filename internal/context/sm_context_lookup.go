package context

import (
	"strings"

	"github.com/free5gc/openapi/models"
)

// GetSMContextBySupi returns the first SMContext matching SUPI and optional DNN/S-NSSAI filters.
func GetSMContextBySupi(supi string, dnn string, snssai *models.Snssai) *SMContext {
	if supi == "" {
		return nil
	}
	var matched *SMContext
	smContextPool.Range(func(_, value any) bool {
		smContext, ok := value.(*SMContext)
		if !ok || smContext == nil {
			return true
		}
		if smContext.Supi != supi {
			return true
		}
		if dnn != "" && smContext.Dnn != dnn {
			return true
		}
		if snssai != nil && smContext.SNssai != nil && !snssaiEqualModels(smContext.SNssai, snssai) {
			return true
		}
		if snssai != nil && smContext.SNssai == nil {
			return true
		}
		matched = smContext
		return false
	})
	return matched
}

func snssaiEqualModels(a *models.Snssai, b *models.Snssai) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Sst == b.Sst && strings.EqualFold(a.Sd, b.Sd)
}
