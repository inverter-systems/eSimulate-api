package security

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// ValidatePasswordStrength valida a força de uma senha
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("senha deve ter no mínimo 8 caracteres")
	}
	
	if len(password) > 128 {
		return errors.New("senha deve ter no máximo 128 caracteres")
	}
	
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	var missing []string
	if !hasUpper {
		missing = append(missing, "maiúscula")
	}
	if !hasLower {
		missing = append(missing, "minúscula")
	}
	if !hasNumber {
		missing = append(missing, "número")
	}
	if !hasSpecial {
		missing = append(missing, "símbolo")
	}
	
	if len(missing) > 0 {
		return errors.New("senha deve conter pelo menos uma letra " + strings.Join(missing, ", uma "))
	}
	
	// Verificar senhas comuns
	if isCommonPassword(password) {
		return errors.New("senha muito comum. Escolha uma senha mais segura")
	}
	
	return nil
}

// isCommonPassword verifica se a senha está em uma lista de senhas comuns
func isCommonPassword(password string) bool {
	commonPasswords := []string{
		"password", "12345678", "123456789", "1234567890",
		"qwerty", "abc123", "password1", "admin123",
		"letmein", "welcome", "monkey", "1234567",
		"sunshine", "princess", "football", "iloveyou",
	}
	
	lowerPassword := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lowerPassword == common {
			return true
		}
	}
	
	// Verificar padrões simples (apenas números ou apenas letras)
	onlyNumbers, _ := regexp.MatchString(`^[0-9]+$`, password)
	onlyLetters, _ := regexp.MatchString(`^[a-zA-Z]+$`, password)
	
	if onlyNumbers || onlyLetters {
		return true
	}
	
	return false
}

