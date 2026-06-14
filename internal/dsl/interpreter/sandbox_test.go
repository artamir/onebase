package interpreter_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ivantit66/onebase/internal/dsl/ast"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/dsl/lexer"
	"github.com/ivantit66/onebase/internal/dsl/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseProc(t *testing.T, src string) *ast.ProcedureDecl {
	t.Helper()
	prog, err := parser.New(lexer.New(src, "test.os")).ParseProgram()
	require.NoError(t, err)
	require.NotEmpty(t, prog.Procedures)
	return prog.Procedures[0]
}

// Бесконечный цикл с пустым телом останавливается по wall-clock,
// и Попытка НЕ перехватывает жёсткий стоп.
func TestSandbox_WallClockHardStop(t *testing.T) {
	src := `Процедура Тест()
  Попытка
    Пока Истина Цикл
    КонецЦикла;
  Исключение
    Возврат "поймано";
  КонецПопытки;
  Возврат "вышли";
КонецПроцедуры`
	p := interpreter.SandboxProfile{MaxWallClock: 50 * time.Millisecond}
	var result any
	err := interpreter.New().RunSandboxed(parseProc(t, src), nil, p, &result)
	require.Error(t, err)
	assert.NotEqual(t, "поймано", result)
	assert.NotEqual(t, "вышли", result)
}

// Цикл сверх MaxLoopIters останавливается жёстко, минуя Попытку.
func TestSandbox_LoopItersHardStop(t *testing.T) {
	src := `Процедура Тест()
  Попытка
    н = 0;
    Пока н < 100000000 Цикл
      н = н + 1;
    КонецЦикла;
  Исключение
    Возврат "поймано";
  КонецПопытки;
  Возврат "вышли";
КонецПроцедуры`
	p := interpreter.SandboxProfile{MaxLoopIters: 1000}
	var result any
	err := interpreter.New().RunSandboxed(parseProc(t, src), nil, p, &result)
	require.Error(t, err)
	assert.NotEqual(t, "поймано", result)
	assert.NotEqual(t, "вышли", result)
}

// Без профиля (нулевые лимиты) обычный цикл отрабатывает и возвращает значение.
func TestSandbox_NoProfileNoRegression(t *testing.T) {
	src := `Процедура Тест()
  с = 0;
  Для к = 1 По 1000 Цикл
    с = с + к;
  КонецЦикла;
  Возврат с;
КонецПроцедуры`
	var result any
	err := interpreter.New().RunSandboxed(parseProc(t, src), nil, interpreter.SandboxProfile{}, &result)
	require.NoError(t, err)
	// DSL числа возвращаются как decimal.Decimal — сравниваем через строку.
	assert.Equal(t, "500500", fmt.Sprintf("%v", result))
}
