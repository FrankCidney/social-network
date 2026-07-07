package config

import "testing"

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "development", env: "development", want: true},
		{name: "dev", env: "dev", want: true},
		{name: "local", env: "local", want: true},
		{name: "test", env: "test", want: true},
		{name: "production", env: "production", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APP_ENV", tt.env)

			if got := IsDevelopment(); got != tt.want {
				t.Fatalf("IsDevelopment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecureCookiesEnabledDefaultsToTrue(t *testing.T) {
	if !SecureCookiesEnabled() {
		t.Fatal("SecureCookiesEnabled() = false, want true when no environment is set")
	}
}
