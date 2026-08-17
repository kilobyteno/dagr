package auth

import "testing"

func TestPasswordPolicyValidate(t *testing.T) {
	t.Parallel()

	policy := PasswordPolicy{
		MinLength:        12,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumber:    true,
		RequireSymbol:    true,
	}

	if err := policy.Validate("Short1!"); err == nil {
		t.Fatal("expected length error")
	}
	if err := policy.Validate("alllowercase1!"); err == nil {
		t.Fatal("expected uppercase error")
	}
	if err := policy.Validate("ALLUPPERCASE1!"); err == nil {
		t.Fatal("expected lowercase error")
	}
	if err := policy.Validate("NoNumbersHere!"); err == nil {
		t.Fatal("expected number error")
	}
	if err := policy.Validate("NoSymbolHere1"); err == nil {
		t.Fatal("expected symbol error")
	}
	if err := policy.Validate("ValidPass12!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
