package metrics

import (
	"fmt"
	"sort"
	"strings"
)

func FormatNormalizedMetric(metric NormalizedMetric) string {
	var b strings.Builder

	b.WriteString("SOURCE: ")
	b.WriteString(metric.Source)
	b.WriteByte('\n')

	b.WriteString("METRIC: ")
	b.WriteString(metric.Name)
	b.WriteByte('\n')

	if len(metric.Labels) > 0 {
		keys := make([]string, 0, len(metric.Labels))

		for key := range metric.Labels {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		b.WriteString("LABELS: ")

		for i, key := range keys {
			if i > 0 {
				b.WriteString(", ")
			}

			fmt.Fprintf(&b, "%s=%s", key, metric.Labels[key])
		}

		b.WriteByte('\n')
	}

	if metric.Instant != nil {
		fmt.Fprintf(&b, "VALUE: %g\n", metric.Instant.Value)
		return b.String()
	}

	if metric.Range == nil {
		return b.String()
	}

	fmt.Fprintf(
		&b,
		"RANGE: %s → %s | STEP: %s\n",
		metric.Range.Start.Format("15:04:05"),
		metric.Range.End.Format("15:04:05"),
		metric.Range.Step,
	)

	fmt.Fprintf(
		&b,
		"FIRST: %g | LAST: %g | MIN: %g | MAX: %g\n",
		metric.Range.StartValue,
		metric.Range.EndValue,
		metric.Range.Min,
		metric.Range.Max,
	)

	if metric.Range.Constant {
		b.WriteString("PATTERN: constant\n")
		return b.String()
	}

	fmt.Fprintf(&b, "SAMPLES: %v\n", metric.Range.Samples)

	return b.String()
}
