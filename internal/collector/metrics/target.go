package metrics

import (
	"fmt"
	"net"
	"strconv"
)

type TargetMapping struct {
	Label  string
	Values map[string]string
}

func BuildTargetSelector(
	target string,
	mapping TargetMapping,
) (string, error) {
	if target == "" {
		return "", fmt.Errorf("target is required")
	}

	if mapping.Label == "" {
		return "", fmt.Errorf(
			"prometheus target mapping label is not configured",
		)
	}

	value, ok := mapping.Values[target]
	if !ok {
		return "", fmt.Errorf(
			"prometheus target %q is not mapped",
			target,
		)
	}

	return fmt.Sprintf(
		`%s=%s`,
		mapping.Label,
		strconv.Quote(value),
	), nil
}

func ResolveHostIP(host string) (string, error) {
	ips, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf(
			"resolve target host %q: %w",
			host,
			err,
		)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf(
			"target host %q resolved to no addresses",
			host,
		)
	}

	return ips[0], nil
}
