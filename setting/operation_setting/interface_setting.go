package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// InterfaceSetting holds an operator's choices for the web interface's
// languages. Every field is optional: left empty, the interface behaves as it
// does without this setting (every shipped language, the browser's language
// first, the general custom currency symbol).
type InterfaceSetting struct {
	// Languages lists the interface language codes the language menu offers,
	// in the order it offers them (for example ["en", "fr"]). Empty offers
	// every language the interface ships.
	Languages []string `json:"languages"`
	// DefaultLanguage is the language a visitor sees until they choose one.
	// Empty keeps the language the browser asks for.
	DefaultLanguage string `json:"default_language"`
	// CurrencySymbols maps an interface language code to the symbol shown for
	// a custom currency in that language. A language without an entry uses
	// general_setting.custom_currency_symbol, which the backend also uses.
	CurrencySymbols map[string]string `json:"currency_symbols"`
	// KeepHeaderLTR keeps the top bars laid out left to right in a
	// right-to-left interface language: the logo stays at the left and the
	// actions at the right, while the navigation links and the page keep the
	// language's direction. False mirrors the bars with the page.
	KeepHeaderLTR bool `json:"keep_header_ltr"`
}

var interfaceSetting = InterfaceSetting{
	Languages:       []string{},
	DefaultLanguage: "",
	CurrencySymbols: map[string]string{},
	KeepHeaderLTR:   false,
}

func init() {
	config.GlobalConfig.Register("interface_setting", &interfaceSetting)
}

// GetInterfaceSetting returns the interface language setting.
func GetInterfaceSetting() *InterfaceSetting {
	return &interfaceSetting
}
