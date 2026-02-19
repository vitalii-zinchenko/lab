// Go Runtime Metrics dashboard.
// Shows GC, memory, goroutine, and thread metrics from prometheus/client_golang.
// The extended Go collector (WithGoCollectorRuntimeMetrics(MetricsAll)) must be
// registered in the target process for the GC duration panel to have data.

local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local var  = g.dashboard.variable;
local ts   = g.panel.timeSeries;
local stat = g.panel.stat;
local prom = g.query.prometheus;

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

local statPanel(title, unit, targets, w=6, h=4, x=0, y=0) =
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

// job variable: query-driven, required (no All), sorted
local jobVar =
  var.query.new('job', 'label_values(go_goroutines, job)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)  // 2 = on time range change
  + var.query.withSort(1)     // 1 = alphabetical asc
  + var.query.generalOptions.withLabel('Job');

// ─── Dashboard ───────────────────────────────────────────────────────────────

g.dashboard.new('Go Runtime Metrics')
+ g.dashboard.withUid('go-runtime')
+ g.dashboard.withDescription('Go runtime metrics: GC, memory, goroutines. Select any job that exposes go_* metrics.')
+ g.dashboard.time.withFrom('now-1h')
+ g.dashboard.time.withTo('now')
+ g.dashboard.withRefresh('30s')
+ g.dashboard.withVariables([datasourceVar, jobVar])
+ g.dashboard.withPanels([

  // ── Overview stats (y=0) ─────────────────────────────────────────────────

  statPanel('Goroutines', 'short',
    [target('go_goroutines{job="$job"}', 'goroutines')],
    6, 4, 0, 0),

  statPanel('Threads', 'short',
    [target('go_threads{job="$job"}', 'threads')],
    6, 4, 6, 0),

  statPanel('Heap In Use', 'bytes',
    [target('go_memstats_heap_inuse_bytes{job="$job"}', 'heap inuse')],
    6, 4, 12, 0),

  statPanel('GC Duration p99', 's',
    [target('go_gc_duration_seconds{job="$job", quantile="0.99"}', 'p99')],
    6, 4, 18, 0),

  // ── Goroutines & Threads (y=4) ────────────────────────────────────────────

  tsPanel('Goroutines', 'short',
    [target('go_goroutines{job="$job"}', 'goroutines')],
    12, 8, 0, 4),

  tsPanel('Threads', 'short',
    [target('go_threads{job="$job"}', 'threads')],
    12, 8, 12, 4),

  // ── Memory (y=12) ─────────────────────────────────────────────────────────

  tsPanel('Heap Memory', 'bytes', [
    target('go_memstats_heap_inuse_bytes{job="$job"}', 'heap inuse',  'A'),
    target('go_memstats_heap_alloc_bytes{job="$job"}', 'heap alloc',  'B'),
    target('go_memstats_heap_idle_bytes{job="$job"}',  'heap idle',   'C'),
    target('go_memstats_heap_sys_bytes{job="$job"}',   'heap sys',    'D'),
  ], 12, 8, 0, 12),

  tsPanel('System Memory Breakdown', 'bytes', [
    target('go_memstats_sys_bytes{job="$job"}',           'sys total',   'A'),
    target('go_memstats_stack_sys_bytes{job="$job"}',     'stack sys',   'B'),
    target('go_memstats_mspan_sys_bytes{job="$job"}',     'mspan sys',   'C'),
    target('go_memstats_mcache_sys_bytes{job="$job"}',    'mcache sys',  'D'),
    target('go_memstats_buck_hash_sys_bytes{job="$job"}', 'buck hash',   'E'),
    target('go_memstats_gc_sys_bytes{job="$job"}',        'gc metadata', 'F'),
  ], 12, 8, 12, 12),

  // ── GC (y=20) ─────────────────────────────────────────────────────────────

  // go_gc_duration_seconds is a summary with quantile labels (classic collector).
  // Requires either the default or extended Go collector.
  tsPanel('GC Stop-the-World Duration', 's', [
    target('go_gc_duration_seconds{job="$job", quantile="0.5"}',  'p50', 'A'),
    target('go_gc_duration_seconds{job="$job", quantile="0.75"}', 'p75', 'B'),
    target('go_gc_duration_seconds{job="$job", quantile="0.9"}',  'p90', 'C'),
    target('go_gc_duration_seconds{job="$job", quantile="0.99"}', 'p99', 'D'),
  ], 12, 8, 0, 20),

  tsPanel('Allocation Rate', 'Bps', [
    target(
      'rate(go_memstats_alloc_bytes_total{job="$job"}[$__rate_interval])',
      'alloc/s'),
  ], 12, 8, 12, 20),

])
