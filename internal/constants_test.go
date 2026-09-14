package internal

import (
	"slices"
	"testing"
)

// TestEveryEnabledProviderIsClassified guards a gap that is silent at compile
// time. A provider must appear both in enabledProviderAuth, which lists the
// environment variables its credentials come from, and in exactly one of
// justTokenProviders or userAndPasswordProviders, which is what checkProvider
// consults when counting configured providers.
//
// Registering a provider in only the first list leaves it counting zero, so a
// run configured with nothing but that provider fails with "no providers
// defined" - the provider is invisible rather than broken, and the build says
// nothing.
func TestEveryEnabledProviderIsClassified(t *testing.T) {
	for provider := range enabledProviderAuth {
		justToken := slices.Contains(justTokenProviders, provider)
		userAndPassword := slices.Contains(userAndPasswordProviders, provider)

		switch {
		case justToken && userAndPassword:
			t.Errorf("%s is in both justTokenProviders and userAndPasswordProviders; checkProvider would count it twice", provider)
		case !justToken && !userAndPassword:
			t.Errorf("%s is in enabledProviderAuth but neither justTokenProviders nor userAndPasswordProviders, so checkProvider counts it as zero and it can never be configured", provider)
		}
	}
}

// TestClassifiedProvidersDeclareTheirCredentials is the reverse: a provider
// that checkProvider knows how to count but that declares no credential
// variables would pass validation without anything to authenticate with.
func TestClassifiedProvidersDeclareTheirCredentials(t *testing.T) {
	for _, provider := range slices.Concat(justTokenProviders, userAndPasswordProviders) {
		vars, ok := enabledProviderAuth[provider]
		if !ok {
			t.Errorf("%s is classified for counting but absent from enabledProviderAuth", provider)

			continue
		}

		if len(vars) == 0 {
			t.Errorf("%s declares no credential environment variables", provider)
		}
	}
}

// TestCodebergIsRegistered pins the provider added alongside the Codeberg host,
// so a future refactor cannot quietly drop it from either registry.
func TestCodebergIsRegistered(t *testing.T) {
	vars, ok := enabledProviderAuth[providerNameCodeberg]
	if !ok {
		t.Fatal("Codeberg missing from enabledProviderAuth")
	}

	if !slices.Contains(vars, envCodebergToken) {
		t.Errorf("Codeberg should require %s, got %v", envCodebergToken, vars)
	}

	if !slices.Contains(justTokenProviders, providerNameCodeberg) {
		t.Error("Codeberg should be in justTokenProviders; it authenticates with a token alone")
	}
}
