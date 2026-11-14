package operation_setting

import (
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
)

func TestInterfaceSettingDefaultsAreEmpty(t *testing.T) {
	s := InterfaceSetting{Languages: []string{}, CurrencySymbols: map[string]string{}}
	if len(s.Languages) != 0 || s.DefaultLanguage != "" || len(s.CurrencySymbols) != 0 {
		t.Fatalf("expected an empty setting, got %+v", s)
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
