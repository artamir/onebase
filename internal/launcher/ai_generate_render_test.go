package launcher

import (
	"strings"
	"testing"
)

// TestConfigurator_GeneratePanelWired проверяет, что панель генерации каркаса
// ИИ подключена в подвал конфигуратора: присутствуют ключевые элементы UI и
// вызовы эндпоинтов ai-generate/ai-apply.
//
// Тест подобран из ветки этапа 2b (план 57) и адаптирован под реализацию
// этапа 3: кнопка отклонения (cfggen-reject) была убрана при упрощении панели,
// закрытие выполняется через cfggen-close.
func TestConfigurator_GeneratePanelWired(t *testing.T) {
	html := renderCfgFoot(t)
	for _, sub := range []string{
		"cfggen-panel", "cfggen-prompt", "cfggen-send", "cfggen-apply",
		"configurator/ai-generate", "configurator/ai-apply",
	} {
		if !strings.Contains(html, sub) {
			t.Errorf("в cfg-foot нет %q — панель генерации не подключена", sub)
		}
	}
}
