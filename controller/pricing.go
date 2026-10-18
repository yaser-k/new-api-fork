package controller

import (
	"maps"
	"sort"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if common.StringsContains(item.EnableGroup, "all") {
			filtered = append(filtered, item)
			continue
		}
		for _, group := range item.EnableGroup {
			if _, ok := usableGroup[group]; ok {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	userId, exists := c.Get("id")
	usableGroup := map[string]string{}
	groupRatio := map[string]float64{}
	maps.Copy(groupRatio, ratio_setting.GetGroupRatioCopy())
	var group string
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio_setting.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	retirements := operation_setting.GetModelRetirementsCopy()
	pricing = withoutRetiredPricing(pricing, retirements)
	usableGroup = service.GetUserUsableGroups(group)
	pricing = filterPricingByUsableGroups(pricing, usableGroup)
	// check groupRatio contains usableGroup
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	response := gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        service.GetUserAutoGroup(group),
		"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
	}
	if retiredModels := pricingRetiredModels(retirements); len(retiredModels) > 0 {
		response["retired_models"] = retiredModels
	}
	c.JSON(200, response)
}

type pricingRetiredModel struct {
	ModelName string `json:"model_name"`
	RetiredAt string `json:"retired_at"`
	Successor string `json:"successor"`
	NoticeURL string `json:"notice_url,omitempty"`
	Note      string `json:"note,omitempty"`
}

// pricingRetiredModels lists the retired models for the pricing page, newest
// retirement first, then by name.
func pricingRetiredModels(retirements map[string]operation_setting.ModelRetirement) []pricingRetiredModel {
	retiredModels := make([]pricingRetiredModel, 0, len(retirements))
	for name, retirement := range retirements {
		retiredModels = append(retiredModels, pricingRetiredModel{
			ModelName: name,
			RetiredAt: retirement.RetiredAt,
			Successor: retirement.Successor,
			NoticeURL: retirement.NoticeURL,
			Note:      retirement.Note,
		})
	}
	sort.Slice(retiredModels, func(i, j int) bool {
		if retiredModels[i].RetiredAt != retiredModels[j].RetiredAt {
			return retiredModels[i].RetiredAt > retiredModels[j].RetiredAt
		}
		return retiredModels[i].ModelName < retiredModels[j].ModelName
	})
	return retiredModels
}

// reportedRetiredPricing holds the retired models already reported as still
// priced, so each is logged once per process.
var reportedRetiredPricing sync.Map

// withoutRetiredPricing drops retired models from the live pricing: they are
// listed in retired_models instead. A retired model that is still priced is a
// misconfiguration, reported once in the server log.
func withoutRetiredPricing(pricing []model.Pricing, retirements map[string]operation_setting.ModelRetirement) []model.Pricing {
	if len(retirements) == 0 {
		return pricing
	}
	live := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if _, retired := retirements[item.ModelName]; !retired {
			live = append(live, item)
			continue
		}
		if _, reported := reportedRetiredPricing.LoadOrStore(item.ModelName, struct{}{}); !reported {
			common.SysLog(common.LogText("retired model %q still has live pricing; the pricing page shows it as retired", item.ModelName))
		}
	}
	return live
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccessT(c, "Model ratios reset", nil)
}
