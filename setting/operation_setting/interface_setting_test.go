package operation_setting

import (
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
)

func TestInterfaceSettingDefaultsAreEmpty(t *testing.T) {
	s := InterfaceSetting{Languages: []string{}, CurrencySymbols: map[string]string{}}
	if len(s.Languages) != 0 || s.DefaultLanguage != "" || len(s.CurrencySymbols) != 0 || s.KeepHeaderLTR {
		t.Fatalf("expected an empty setting, got %+v", s)
	}
	if GetInterfaceSetting().KeepHeaderLTR {
		t.Fatal("keep_header_ltr must default to false so the bars mirror as before")
	}
}

func TestInterfaceSettingKeepHeaderLTRLoadsFromOptions(t *testing.T) {
	s := InterfaceSetting{Languages: []string{}, CurrencySymbols: map[string]string{}}
	if err := config.UpdateConfigFromMap(&s, map[string]string{"keep_header_ltr": "true"}); err != nil {
		t.Fatalf("UpdateConfigFromMap: %v", err)
	}
	if !s.KeepHeaderLTR {
		t.Fatal("keep_header_ltr = false after the option was set to true")
	}
	if err := config.UpdateConfigFromMap(&s, map[string]string{"keep_header_ltr": "false"}); err != nil {
		t.Fatalf("UpdateConfigFromMap: %v", err)
	}
	if s.KeepHeaderLTR {
		t.Fatal("keep_header_ltr = true after the option was set back to false")
	}
}

func TestInterfaceSettingLoadsFromOptions(t *testing.T) {
	s := InterfaceSetting{Languages: []string{}, CurrencySymbols: map[string]string{"xx": "old"}}
	err := config.UpdateConfigFromMap(&s, map[string]string{
		"languages":        `["fr","en"]`,
		"default_language": "fr",
		"currency_symbols": `{"fr":"€","en":"EUR"}`,
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromMap: %v", err)
	}
	if !reflect.DeepEqual(s.Languages, []string{"fr", "en"}) {
		t.Errorf("languages = %v", s.Languages)
	}
	if s.DefaultLanguage != "fr" {
		t.Errorf("default_language = %q", s.DefaultLanguage)
	}
	want := map[string]string{"fr": "€", "en": "EUR"}
	if !reflect.DeepEqual(s.CurrencySymbols, want) {
		t.Errorf("currency_symbols = %v, want %v (a new map replaces the old one)", s.CurrencySymbols, want)
	}
}

func TestInterfaceSettingIsRegistered(t *testing.T) {
	if config.GlobalConfig.Get("interface_setting") == nil {
		t.Fatal("interface_setting is not registered with the config manager")
	}
}
