package luhn

import (
	"strconv"
	"strings"
	"unicode"
)

// Validate проверяет, соответствует ли номер алгоритму Луна
func Validate(number string) bool {
	// Удаляем все пробелы из строки
	number = strings.ReplaceAll(number, " ", "")

	// Проверяем, что строка состоит только из цифр
	for _, r := range number {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	// Проверяем минимальную длину (обычно не менее 2 цифр)
	if len(number) < 2 {
		return false
	}

	// Преобразуем строку в слайс цифр
	digits := make([]int, len(number))
	for i, r := range number {
		digit, err := strconv.Atoi(string(r))
		if err != nil {
			return false
		}
		digits[i] = digit
	}

	// Проходим по цифрам справа налево
	sum := 0
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		// Каждую вторую цифру удваиваем
		if (len(digits)-i)%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit = digit - 9
			}
		}
		sum += digit
	}

	// Сумма должна быть кратна 10
	return sum%10 == 0
}
