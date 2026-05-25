package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Регрессия: UI разрешает теперь сохранить пустой пароль (kiosk/тестовый
// аккаунт). Проверяем, что:
//  1. bcrypt.GenerateFromPassword корректно хеширует пустую строку,
//  2. bcrypt.CompareHashAndPassword принимает пустой ввод против такого
//     хеша,
//  3. непустой ввод не проходит против пустого хеша (защита от случайного
//     «всё подходит»).
//
// Если на каком-то этапе bcrypt начнёт ругаться на len(password)==0 —
// тест упадёт и мы вернём минимальную проверку в handler-ы.
func TestBcrypt_EmptyPasswordRoundtrip(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte(""), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword(\"\"): %v — пустые пароли больше не поддерживаются bcrypt-ом", err)
	}
	if len(hash) == 0 {
		t.Fatal("хеш пустой строки сам оказался пустым")
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("")); err != nil {
		t.Errorf("Compare(hash(\"\"), \"\") должен пройти, получили: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("notempty")); err == nil {
		t.Error("Compare(hash(\"\"), \"notempty\") должен ОТКЛОНИТЬ — иначе любой ввод подходит к пустому паролю")
	}
}
