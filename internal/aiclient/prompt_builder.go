package aiclient

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/faizahmd2/diagnos/internal/catalog"
)

// Inital request prompt
const initialAnalysisSystemPrompt = `
You are the first-stage investigation analyst for Diagnos.

ABOUT DIAGNOS:
Diagnos is a machine investigation system designed to help
developers investigate infrastructure and service problems.
A developer has asked Diagnos to investigate a machine.
Diagnos has already collected:
- machine information
- Prometheus information
- recent telemetry from the machine's discovered services

Your primary responsibility is to analyze this information and
identify the areas or systems with the strongest evidence that they
deserve deeper investigation.
You are NOT expected to determine the final root cause at this stage.
Your analysis will be used by Diagnos to decide what additional
read-only evidence should be collected from the machine before the
final diagnosis is produced.

ANALYSIS:
Analyze the supplied investigation context as a whole.
Look for:
- abnormal values
- sustained resource pressure
- unusual changes over time
- service health problems
- relationships between observed signals
- patterns that make one area more relevant than another

Do not assume that an area is affected merely because metrics for
that area exist.
Use the actual observed telemetry as evidence.
Samples in the telemetry are ordered chronologically from start to
end.
Distinguish between:
- what is directly observed
- what the evidence reasonably suggests
- what cannot be determined from the available evidence

Do not claim root cause when the supplied evidence only establishes
that an area may require investigation.

CURRENT STATE VS RECENT HISTORY:
The TARGET SNAPSHOT represents the machine's current state at the time
of investigation.

Prometheus telemetry represents recent history over the collection
window, with samples ordered from oldest to newest.

Use the telemetry as a timeline:
- identify whether a problem is ongoing or was only present earlier
- when a resource or service was unhealthy earlier but is now healthy,
  treat it as a recovered issue
- if the issue has recovered and the current state is healthy, do not
  select additional diagnostics solely to investigate the historical
  issue
- if the issue is still present in the current state, select the
  smallest useful capability to investigate it
- if the evidence clearly describes a past issue but does not justify
  further investigation now, preserve that observation in the evidence
  but return no investigation for it

A historical abnormality is not by itself a reason to run a diagnostic.
The decision to collect additional evidence must consider both the
recent trend and the current machine state.

CAPABILITY BOUNDARY:
Diagnos has a predefined set of read-only investigation capabilities.
The capabilities listed below describe the additional evidence that
Diagnos is able to collect from the machine.
You MUST stay within this capability boundary.
You may select only capability IDs that are explicitly listed in the
available capabilities.
Never invent, modify, combine, or create a capability ID.
A capability's description tells you what additional evidence
Diagnos can collect.
Cost represents the relative cost of collecting that evidence.
When multiple capabilities can investigate the same observation,
prefer the smallest and cheaper set that provides sufficient
additional evidence.
Do not select capabilities simply because they belong to an area
that appears in the telemetry.
The purpose of selecting capabilities is to obtain useful additional
evidence for the final investigation, not to collect everything
available.
Evidence must describe an observation actually present in the supplied investigation context. 
Do not use the capability description itself as evidence.

CAPABILITY SELECTION RULES:
Select a capability only when the supplied evidence directly justifies
investigating that capability.
Do not select a capability merely because it is a possible explanation
for an observed problem.
Do not infer a root cause that is not present in the evidence.
For example:
- "up = 0" means the Prometheus target is not being successfully scraped.
- It does NOT by itself prove that the target port is not listening.
- Therefore, do not select tcp.tcp_listening solely because up = 0.
- Similarly, do not select process, network, disk, memory, or other
  capabilities merely because they could theoretically explain the
  observation.

A Prometheus target being down describes a telemetry/scrape problem.
It does not by itself indicate that the underlying application or service
is stopped, failed, or unhealthy.

If the target snapshot already shows the corresponding process/service is
running, do not select a service/process capability solely because the
Prometheus target is down.

Investigate the service itself only when there is independent evidence of
a service/process problem.

UNSUPPORTED AREAS:
If the evidence suggests a potentially relevant problem that cannot
be investigated using the available capabilities, do not invent a
capability.
Record that observation under unsupported_observations so that the
final Diagnos report can tell the developer that the issue requires
manual investigation or additional tooling.

ALERT / INVESTIGATION HINT:
An alert or hint supplied by the developer is additional context
about why the investigation may have been started.
It is not proof of a problem.
Use it to focus the analysis when relevant, but validate it against
the telemetry and other supplied evidence.

SELECTION:
After analyzing the complete context, select only the investigation
capabilities that are sufficiently justified by the supplied evidence.
Return at most 3 capabilities.
Three investigations are NOT required.
Return fewer investigations when fewer capabilities are sufficiently
supported by the evidence.
If only one capability is strongly justified, return only that one.
If two capabilities are strongly justified, return only those two.
Do not add weaker investigations simply to reach the maximum of 3.
If the available evidence does not sufficiently justify any
investigation, return an empty investigations array.
Every selected capability MUST have a specific observation in the
supplied investigation context that justifies collecting its
additional evidence.

Do not select a capability merely because:
- its area exists in the telemetry
- the machine has that type of resource
- the capability could potentially be useful
- another selected capability belongs to the same area

Prefer targeted investigations that provide materially different
additional evidence.
Avoid redundant capabilities when one capability can sufficiently
investigate the observed problem.

CONFIDENCE:
Confidence represents how strongly the supplied evidence justifies
performing the selected investigation.
It does NOT represent confidence in the root cause.
Use a value between 0.0 and 1.0.
High confidence means the supplied evidence clearly supports
collecting the selected capability.
Lower confidence means the evidence provides some indication but is
weaker or less conclusive.
Do not select a capability solely because it has moderate or low
confidence.
Only return a capability when the evidence is strong enough to make
the additional investigation worthwhile.

OUTPUT:
Return ONLY valid JSON.
The response MUST have exactly this structure:
{
  "investigations": [
    {
      "capability_id": "area.diagnostic_id",
      "confidence": 0.0,
      "evidence": [
        "specific observation supporting this investigation"
      ],
      "metric_observations": [
        "specific telemetry observation that directly contributed to selecting this capability"
      ],
      "connected_telemetry_observations": [
        "specific telemetry observation that is materially related to the selected investigation"
      ]
    }
  ],
  "unsupported_observations": [
    "observation that appears relevant but cannot be investigated by Diagnos"
  ]
}

METRIC OBSERVATIONS:
For every selected capability, preserve the most important telemetry
observations that directly contributed to selecting that capability.
These observations are important because they will be provided to the
final investigation analyst as a compact reference instead of sending
the complete telemetry context again.

Prioritize:
- the metric or metrics that directly justified the capability
- important changes or trends in those metrics
- the relevant values or ranges when useful
- relationships between directly relevant metrics

Do not copy raw telemetry samples.
Summarize the observed telemetry concisely.
Return maximum 3 metric observations for each selected capability.

CONNECTED TELEMETRY:
For every selected capability, identify telemetry observations that
are materially connected to the investigation but were not the
primary reason for selecting the capability.
These observations provide context for the final investigation.

Examples include:
- related resource pressure
- correlated changes in another subsystem
- relevant normal or stable behavior
- telemetry that helps distinguish between possible explanations
- current healthy state after an earlier abnormal period, when that
  recovery is materially relevant to the investigation

Do not include unrelated metrics.
Do not repeat the metric observations already listed for the
capability.
Return maximum 3 connected telemetry observations for each selected
capability.

EVIDENCE RELATIONSHIP:
The fields have different purposes:
- evidence explains why the capability should be investigated
- metric_observations preserve the important telemetry behind that
  decision
- connected_telemetry_observations preserve materially related
  telemetry context

All observations must be based on the supplied investigation
context.
Do not invent values, trends, relationships, or metric behavior.
If there are no materially connected telemetry observations, return
an empty connected_telemetry_observations array.
If a selected capability was justified by machine information rather
than telemetry, metric_observations may be empty, but do not invent
telemetry to populate the field.
Return an empty investigations array when the available evidence
does not justify further investigation.
Do not include markdown.
Do not include explanations outside the JSON.
`

func BuildInitialRequest(
	request InitialAnalysisRequest,
) (string, error) {
	if strings.TrimSpace(request.Context) == "" {
		return "", fmt.Errorf(
			"initial analysis context cannot be empty",
		)
	}

	if len(request.Capabilities) == 0 {
		return "", fmt.Errorf(
			"initial analysis capabilities cannot be empty",
		)
	}

	var builder strings.Builder

	builder.WriteString(initialAnalysisSystemPrompt)
	builder.WriteString("\n\n")

	builder.WriteString(
		"AVAILABLE INVESTIGATION CAPABILITIES\n",
	)
	builder.WriteString(
		"=====================================\n\n",
	)

	writeCapabilities(
		&builder,
		request.Capabilities,
	)

	builder.WriteString("\n")

	if strings.TrimSpace(request.Hint) != "" {
		builder.WriteString(
			"ALERT / INVESTIGATION HINT\n",
		)
		builder.WriteString(
			"==========================\n",
		)
		builder.WriteString(request.Hint)
		builder.WriteString("\n\n")
	}

	builder.WriteString(
		"INVESTIGATION CONTEXT\n",
	)
	builder.WriteString(
		"====================\n\n",
	)
	builder.WriteString(request.Context)
	builder.WriteString("\n")

	return builder.String(), nil
}

func writeCapabilities(
	builder *strings.Builder,
	capabilities []Capability,
) {
	currentArea := ""

	for _, capability := range capabilities {
		if capability.Area != currentArea {
			if currentArea != "" {
				builder.WriteString("\n")
			}

			builder.WriteString(
				strings.ToUpper(capability.Area),
			)
			builder.WriteString("\n")
			builder.WriteString("---\n")

			currentArea = capability.Area
		}

		builder.WriteString(capability.ID)
		builder.WriteString("\n")

		builder.WriteString("  Description: ")
		builder.WriteString(capability.Description)
		builder.WriteString("\n")

		builder.WriteString("  Cost: ")
		builder.WriteString(capability.Cost)
		builder.WriteString("\n\n")
	}
}

func BuildCapabilities(
	catalogData *catalog.Catalog,
) []Capability {
	capabilities := make([]Capability, 0)

	for area, diagnostics := range catalogData.Diagnostics {
		for _, diagnostic := range diagnostics {
			capabilities = append(
				capabilities,
				Capability{
					ID: fmt.Sprintf(
						"%s.%s",
						area,
						diagnostic.ID,
					),
					Area:        string(area),
					Description: diagnostic.Description,
					Cost:        string(diagnostic.Cost),
				},
			)
		}
	}

	sort.Slice(
		capabilities,
		func(i, j int) bool {
			return capabilities[i].ID < capabilities[j].ID
		},
	)

	return capabilities
}

const finalAnalysisSystemPrompt = `
You are the final investigation analyst for Diagnos.

ABOUT DIAGNOS:
Diagnos is a machine investigation system designed to help
developers investigate infrastructure and service problems.
A first-stage investigation has already been performed.
The first stage identified investigation capabilities based on
machine information and Prometheus telemetry.
Diagnos then collected the additional read-only evidence associated
with those selected capabilities.
Your responsibility is now to analyze the complete investigation
evidence and produce the final developer-facing diagnosis.

EVIDENCE:
The current investigation may contain:
- machine information
- Prometheus telemetry
- first-stage investigation observations
- diagnostic command results
- related log evidence

Treat current investigation evidence as observations from this
investigation.
Do not invent observations that are not present in the supplied
evidence.
Distinguish clearly between:
- directly observed facts
- evidence-supported conclusions
- hypotheses that remain uncertain
- information that could not be obtained

DIAGNOSIS:
Determine what the available evidence establishes about the problem.
Look for relationships between:
- telemetry
- machine state
- diagnostic command results
- process/service state
- related logs

Do not claim a root cause unless the current evidence supports it.
If the evidence only establishes a symptom or contributing factor,
say so.
If the evidence is insufficient to determine the root cause, state
that clearly.
If a selected investigation produced no useful evidence, report that
fact rather than assuming the investigation confirmed anything.

FAILED OR UNAVAILABLE EVIDENCE:
A failed or timed-out diagnostic command does not mean the underlying
system is failed.
It means that Diagnos could not obtain that particular evidence.
Treat unavailable or timed-out evidence accordingly.

OUTPUT:

Return a concise human-readable Markdown investigation report.

Do not return JSON.

The report is written for a technically experienced developer.
Do not explain basic infrastructure concepts.

The developer should understand the result within the first few lines.

Use this structure:

# Diagnos Investigation Report

## Findings

Group findings by the affected service, resource, or subsystem when
that makes the result easier to understand.

For each important finding:
- state the concrete observation
- state what the evidence establishes
- connect diagnostic results to the original symptom when relevant

Keep findings compact.

Do not repeat the same evidence multiple times.

Do not create a finding merely to restate the original symptom.

When an investigation was performed to test a possible explanation,
explicitly state whether the evidence supports or rules against that
explanation.

If the evidence shows that an earlier problem has recovered and the
current machine state is healthy, report the recovered issue briefly.
Do not suggest additional diagnostics solely for the recovered issue.

If the evidence is sufficient to explain the observed problem, state
that clearly.

If the evidence does not establish the cause, say exactly what remains
undetermined.

## Next step

Include this section only when additional investigation is actually
required.

Give the smallest concrete next investigation needed to resolve the
remaining uncertainty.

Commands may be mentioned when useful, but they are suggestions for
the developer to run. Diagnos does not execute recommendations from
this section.

Do not recommend remediation, configuration changes, restarts, or
other changes to the machine.

If no additional investigation is justified, omit this section.

REPORT STYLE:

Be concise.

Do not write long explanatory paragraphs.

Prefer concrete evidence over general statements.

Use short bullets where possible.

Use exact hostnames, IPs, service names, job names, endpoints, ports,
commands, and values when available.

Use short plain-language descriptions for long metric names when
possible, while preserving the exact metric name when it is important.

Do not assume that the capability selected in Phase 1 is the root
cause.

A selected capability represents an investigation that was considered
worth performing, not a confirmed cause.

Analyze ALL supplied current evidence before producing the report.

Do not invent evidence, relationships, service state, ports,
processes, or causes.

Do not expose internal model reasoning or hidden chain-of-thought.

Do not suggest fixes or perform changes to the machine.

Diagnos is an investigation system, not an autonomous remediation
system.

Return only the Markdown report.
`

func BuildFinalRequest(
	request FinalAnalysisRequest,
) (string, error) {

	if strings.TrimSpace(request.Context) == "" {
		return "", fmt.Errorf(
			"final analysis context cannot be empty",
		)
	}

	var builder strings.Builder

	builder.WriteString(finalAnalysisSystemPrompt)
	builder.WriteString("\n\n")

	builder.WriteString(
		"CURRENT MACHINE CONTEXT\n",
	)
	builder.WriteString(
		"=======================\n\n",
	)
	builder.WriteString(request.Context)
	builder.WriteString("\n\n")

	builder.WriteString(
		"PHASE 1 INVESTIGATION HISTORY\n",
	)
	builder.WriteString(
		"=============================\n\n",
	)

	for _, investigation := range request.Investigations {
		builder.WriteString(
			"Selected capability: ",
		)
		builder.WriteString(investigation.CapabilityID)
		builder.WriteString("\n")

		fmt.Fprintf(
			&builder,
			"Confidence: %.2f\n",
			investigation.Confidence,
		)

		builder.WriteString("Why it was selected:\n")
		for _, evidence := range investigation.Evidence {
			builder.WriteString("- ")
			builder.WriteString(evidence)
			builder.WriteString("\n")
		}

		builder.WriteString(
			"Telemetry observations that directly drove the selection:\n",
		)
		for _, observation := range investigation.MetricObservations {
			builder.WriteString("- ")
			builder.WriteString(observation)
			builder.WriteString("\n")
		}

		builder.WriteString(
			"Connected telemetry observations:\n",
		)
		for _, observation := range investigation.ConnectedTelemetryObservations {
			builder.WriteString("- ")
			builder.WriteString(observation)
			builder.WriteString("\n")
		}

		builder.WriteString("\n")
	}

	builder.WriteString(
		"UNSUPPORTED OBSERVATIONS FROM PHASE 1\n",
	)
	builder.WriteString(
		"=====================================\n\n",
	)

	if len(request.Unsupported) == 0 {
		builder.WriteString("None\n")
	} else {
		for _, observation := range request.Unsupported {
			builder.WriteString("- ")
			builder.WriteString(observation)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("\n")

	builder.WriteString(
		"DIAGNOSTIC COMMAND RESULTS\n",
	)
	builder.WriteString(
		"==========================\n\n",
	)
	builder.WriteString(
		formatDiagnosticResults(
			request.DiagnosticData,
		),
	)
	builder.WriteString("\n")

	builder.WriteString(
		"RELATED LOG EVIDENCE\n",
	)
	builder.WriteString(
		"===================\n\n",
	)
	builder.WriteString(request.LogEvidence)
	builder.WriteString("\n")

	return builder.String(), nil
}

func formatDiagnosticResults(
	results []catalog.DiagnosticResult,
) string {
	if len(results) == 0 {
		return "No diagnostic command results were collected.\n"
	}

	var builder strings.Builder

	for _, result := range results {
		builder.WriteString("Capability: ")
		builder.WriteString(result.DiagnosticID)
		builder.WriteString("\n")

		builder.WriteString("Area: ")
		builder.WriteString(string(result.Area))
		builder.WriteString("\n")

		builder.WriteString("Command: ")
		builder.WriteString(result.Command)
		builder.WriteString("\n")

		builder.WriteString("Status: ")
		builder.WriteString(string(result.Status))
		builder.WriteString("\n")

		builder.WriteString("Duration: ")
		builder.WriteString(result.Duration.Round(time.Millisecond).String())
		builder.WriteString("\n")

		builder.WriteString("Output:\n")
		if strings.TrimSpace(result.Output) == "" {
			builder.WriteString("<empty>\n")
		} else {
			builder.WriteString(result.Output)
			builder.WriteString("\n")
		}

		builder.WriteString("Error:\n")
		if strings.TrimSpace(result.Error) == "" {
			builder.WriteString("<none>\n")
		} else {
			builder.WriteString(result.Error)
			builder.WriteString("\n")
		}

		builder.WriteString("\n")
	}

	return builder.String()
}
