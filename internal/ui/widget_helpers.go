package ui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/ivantit66/onebase/internal/widget"
)

// widgetCell formats a single cell in a list-widget table according to the
// declared format (money/number/percent/date). It is wired into the template
// FuncMap as "wcell" and called like `{{wcell row "Field" "money"}}`.
func widgetCell(row map[string]any, field, format string) string {
	v, ok := row[field]
	if !ok || v == nil {
		return ""
	}
	switch strings.ToLower(format) {
	case "money":
		return formatMoneyForCell(v)
	case "number":
		return formatIntForCell(v)
	case "percent":
		f := toFloatForCell(v)
		return fmt.Sprintf("%.1f%%", f)
	case "date":
		if t, ok := v.(time.Time); ok {
			return t.Format("02.01.2006 15:04")
		}
		return fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf("%v", v)
}

// echartsJSON serializes ChartData into an ECharts option payload, ready for
// JSON.parse on the client side. Returned as template.JS so html/template
// preserves the JSON unchanged inside <script>; the wrapping template emits it
// as a JavaScript expression, not an attribute, which avoids quote-escaping
// pitfalls.
func echartsJSON(chart *widget.ChartData) template.JS {
	if chart == nil {
		return template.JS("null")
	}
	opt := map[string]any{
		"tooltip": map[string]any{"trigger": "axis"},
		"grid":    map[string]any{"left": 40, "right": 16, "top": 24, "bottom": 30},
	}
	switch strings.ToLower(chart.Kind) {
	case "pie":
		var data []map[string]any
		if len(chart.Series) > 0 {
			s := chart.Series[0]
			for i, label := range chart.XAxis {
				if i >= len(s.Data) {
					break
				}
				data = append(data, map[string]any{"name": label, "value": s.Data[i]})
			}
		}
		opt["tooltip"] = map[string]any{"trigger": "item"}
		opt["series"] = []map[string]any{{
			"type":   "pie",
			"radius": []string{"40%", "70%"},
			"data":   data,
		}}
	default:
		seriesType := "bar"
		if strings.EqualFold(chart.Kind, "line") {
			seriesType = "line"
		}
		var series []map[string]any
		for _, s := range chart.Series {
			series = append(series, map[string]any{
				"name":   s.Name,
				"type":   seriesType,
				"data":   s.Data,
				"smooth": seriesType == "line",
			})
		}
		opt["xAxis"] = map[string]any{"type": "category", "data": chart.XAxis}
		opt["yAxis"] = map[string]any{"type": "value"}
		opt["series"] = series
	}
	b, err := json.Marshal(opt)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}

func toFloatForCell(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		var f float64
		fmt.Sscanf(t, "%f", &f)
		return f
	}
	return 0
}

func formatMoneyForCell(v any) string {
	f := toFloatForCell(v)
	neg := f < 0
	if neg {
		f = -f
	}
	whole := int64(f)
	frac := int64((f - float64(whole)) * 100)
	if frac < 0 {
		frac = -frac
	}
	s := groupThousands(whole)
	out := fmt.Sprintf("%s,%02d ₽", s, frac)
	if neg {
		out = "-" + out
	}
	return out
}

func formatIntForCell(v any) string {
	f := toFloatForCell(v)
	neg := f < 0
	if neg {
		f = -f
	}
	whole := int64(f + 0.5)
	s := groupThousands(whole)
	if neg {
		return "-" + s
	}
	return s
}

func groupThousands(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteByte(' ')
		}
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(' ')
		}
	}
	return b.String()
}
