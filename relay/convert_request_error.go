package relay

import (
	"errors"
	"fmt"
	"net/http"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	kitreasoning "github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

func newConvertRequestFailedError(c *gin.Context, info *relaycommon.RelayInfo, err error) *types.NewAPIError {
	var loss *types.ConversionLossError
	if errors.As(err, &loss) {
		info.RecordConversionDiagnostics(c, loss.Diagnostics)
		return types.NewErrorWithStatusCode(err, types.ErrorCodeConvertRequestFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if kitreasoning.IsClientError(err) {
		return types.NewErrorWithStatusCode(err, types.ErrorCodeConvertRequestFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
}

// newUnsupportedOperationError answers a request whose channel adaptor does not
// implement the operation at all (its Convert...Request returns "not
// implemented") as the client's mistake, naming the model and the requested
// path rather than the adaptor. It returns nil for any other conversion error,
// which keeps the caller's own classification.
func newUnsupportedOperationError(c *gin.Context, info *relaycommon.RelayInfo, err error) *types.NewAPIError {
	if err == nil || err.Error() != "not implemented" {
		return nil
	}
	return types.NewErrorWithStatusCode(fmt.Errorf("model %s does not support %s", info.OriginModelName, c.Request.URL.Path), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}
