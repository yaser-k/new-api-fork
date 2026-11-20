package model

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
)

// WithheldBillingExprError returns the error a client receives in place of
// err, a failure to run the billing expression of modelName, when that
// expression is internal by the rule consume rows follow: a task passes the
// model it maps to as mappedModel, any other request passes "". The engine's
// message quotes the expression, so the client is told only which model's
// price failed and err goes to the server log. It returns nil when the
// expression is not internal; the caller then returns err as it always has.
func WithheldBillingExprError(ctx context.Context, modelName, mappedModel string, err error) error {
	if !billingExprInternal(modelName, mappedModel) {
		return nil
	}
	logger.LogError(ctx, common.LogText("model %s tiered expr run failed: %s", modelName, err.Error()))
	return fmt.Errorf("model %s tiered expr run failed", modelName)
}
