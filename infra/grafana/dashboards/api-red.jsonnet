// API RED Metrics dashboard.
// Shows Rate, Errors, and Duration for the Go/Gin API server.
//
// Metric names use the "gin" namespace configured in main.go:
//   ginprometheus.NewPrometheus("gin")
//
// Variables:
//   datasource  — Prometheus datasource (required)
//   job         — Prometheus job label (required, no All)
//   endpoint    — URL/route filter (optional, multi-select, defaults to All)
//
// All time-series panels always break down by endpoint (url label) so you can
// see per-endpoint contribution even when the endpoint variable is set to All.

local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local var  = g.dashboard.variable;
local ts   = g.panel.timeSeries;
local stat = g.panel.stat;
local prom = g.query.prometheus;

// Metric names — "gin" prefix from NewPrometheus("gin") in main.go
local reqTotal    = 'gin_requests_total';
local reqDuration = 'gin_request_duration_seconds';

// ─── Query helper ────────────────────────────────────────────────────────────

local target(expr, legendFormat, refId='A') =
  prom.new('${datasource}', expr)
  + prom.withLegendFormat(legendFormat)
  + prom.withRefId(refId);

// ─── Panel helpers ───────────────────────────────────────────────────────────

local tsPanel(title, unit, targets, w=12, h=8, x=0, y=0) =
  ts.new(title)
  + ts.standardOptions.withUnit(unit)
  + ts.queryOptions.withTargets(targets)
  + ts.gridPos.withH(h)
  + ts.gridPos.withW(w)
  + ts.gridPos.withX(x)
  + ts.gridPos.withY(y)
  + ts.options.tooltip.withMode('multi');

local statPanel(title, unit, targets, w=8, h=4, x=0, y=0) =
  stat.new(title)
  + stat.standardOptions.withUnit(unit)
  + stat.queryOptions.withTargets(targets)
  + stat.gridPos.withH(h)
  + stat.gridPos.withW(w)
  + stat.gridPos.withX(x)
  + stat.gridPos.withY(y)
  + stat.options.reduceOptions.withCalcs(['lastNotNull'])
  + stat.options.withColorMode('value')
  + stat.options.withGraphMode('none');

// ─── Variables ───────────────────────────────────────────────────────────────

local datasourceVar =
  var.datasource.new('datasource', 'prometheus')
  + var.datasource.generalOptions.withLabel('Data Source');

// job: required, no All option
local jobVar =
  var.query.new('job', 'label_values(' + reqTotal + ', job)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)
  + var.query.withSort(1)
  + var.query.generalOptions.withLabel('Job');

// method: optional, multi-select, All = regex .* (matches every method label)
local methodVar =
  var.query.new('method', 'label_values(' + reqTotal + '{job="$job"}, method)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)
  + var.query.withSort(1)
  + var.query.selectionOptions.withIncludeAll(true)
  + var.query.selectionOptions.withMulti(true)
  + var.query.generalOptions.withLabel('Method');

// endpoint: optional, multi-select, All = regex .* (matches every url label)
// filtered by selected method so the dropdown only shows relevant paths.
local endpointVar =
  var.query.new('endpoint', 'label_values(' + reqTotal + '{job="$job", method=~"$method"}, url)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)
  + var.query.withSort(1)
  + var.query.selectionOptions.withIncludeAll(true)
  + var.query.selectionOptions.withMulti(true)
  + var.query.generalOptions.withLabel('Endpoint');

// ─── Dashboard ───────────────────────────────────────────────────────────────

g.dashboard.new('API RED Metrics')
+ g.dashboard.withUid('api-red')
+ g.dashboard.withDescription('Rate, Errors, Duration for the Go/Gin API. Select job (required) and optionally filter by endpoint.')
+ g.dashboard.time.withFrom('now-1h')
+ g.dashboard.time.withTo('now')
+ g.dashboard.withRefresh('30s')
+ g.dashboard.withVariables([datasourceVar, jobVar, methodVar, endpointVar])
+ g.dashboard.withPanels([

  // ── Summary stats (y=0) ───────────────────────────────────────────────────

  statPanel('Request Rate', 'reqps', [
    target(
      'sum(rate(' + reqTotal + '{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval]))',
      'req/s'),
  ], 8, 4, 0, 0),

  // Error rate as a ratio 0-1 (shown as % via percentunit).
  // Uses "or vector(0)" so the panel shows 0 instead of "No data" when there
  // are no 5xx errors in the selected window.
  statPanel('Error Rate', 'percentunit', [
    target(
      '(sum(rate(' + reqTotal + '{job="$job", method=~"$method", url=~"$endpoint", code=~"5.."}[$__rate_interval])) or vector(0))'
      + ' / sum(rate(' + reqTotal + '{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval]))',
      'error rate'),
  ], 8, 4, 8, 0),

  statPanel('p99 Latency', 's', [
    target(
      'histogram_quantile(0.99, sum by (le) (rate(' + reqDuration + '_bucket{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval])))',
      'p99'),
  ], 8, 4, 16, 0),

  // ── Rate: requests/sec broken down by method and endpoint (y=4) ──────────

  tsPanel('Request Rate by Endpoint', 'reqps', [
    target(
      'sum by (method, url) (rate(' + reqTotal + '{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval]))',
      '{{method}} {{url}}'),
  ], 24, 8, 0, 4),

  // ── Errors: 5xx/sec broken down by method and endpoint (y=12) ────────────

  tsPanel('Error Rate by Endpoint (5xx req/s)', 'reqps', [
    target(
      'sum by (method, url) (rate(' + reqTotal + '{job="$job", method=~"$method", url=~"$endpoint", code=~"5.."}[$__rate_interval]))',
      '{{method}} {{url}}'),
  ], 24, 8, 0, 12),

  // ── Duration (y=20) ───────────────────────────────────────────────────────

  // Aggregate latency percentiles across all selected endpoints
  tsPanel('Latency Percentiles (aggregate)', 's', [
    target(
      'histogram_quantile(0.50, sum by (le) (rate(' + reqDuration + '_bucket{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval])))',
      'p50', 'A'),
    target(
      'histogram_quantile(0.95, sum by (le) (rate(' + reqDuration + '_bucket{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval])))',
      'p95', 'B'),
    target(
      'histogram_quantile(0.99, sum by (le) (rate(' + reqDuration + '_bucket{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval])))',
      'p99', 'C'),
  ], 12, 8, 0, 20),

  // p99 per method+endpoint — shows which endpoint is the slowest
  tsPanel('p99 Latency by Endpoint', 's', [
    target(
      'histogram_quantile(0.99, sum by (le, method, url) (rate(' + reqDuration + '_bucket{job="$job", method=~"$method", url=~"$endpoint"}[$__rate_interval])))',
      '{{method}} {{url}}'),
  ], 12, 8, 12, 20),

])
