// PostgreSQL Overview dashboard.
// Metrics sourced from prometheus-community/postgres_exporter (port 9187).
// Covers connections, transaction rates, cache hit ratio, locks, and deadlocks.

local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local var  = g.dashboard.variable;
local ts   = g.panel.timeSeries;
local stat = g.panel.stat;
local prom = g.query.prometheus;

// ─── Query helpers ───────────────────────────────────────────────────────────

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

local statPanel(title, unit, targets, w=6, h=4, x=0, y=0, thresholds=[]) =
  stat.new(title)
  + stat.standardOptions.withUnit(unit)
  + stat.queryOptions.withTargets(targets)
  + stat.gridPos.withH(h)
  + stat.gridPos.withW(w)
  + stat.gridPos.withX(x)
  + stat.gridPos.withY(y)
  + stat.options.reduceOptions.withCalcs(['lastNotNull'])
  + stat.options.withColorMode('value')
  + stat.options.withGraphMode('none')
  + (if std.length(thresholds) > 0 then
       stat.standardOptions.thresholds.withMode('absolute')
       + stat.standardOptions.thresholds.withSteps(thresholds)
     else {});

// ─── Variables ───────────────────────────────────────────────────────────────

local datasourceVar =
  var.datasource.new('datasource', 'prometheus')
  + var.datasource.generalOptions.withLabel('Data Source');

local jobVar =
  var.query.new('job', 'label_values(pg_stat_database_numbackends, job)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)
  + var.query.withSort(1)
  + var.query.generalOptions.withLabel('Job');

local databaseVar =
  var.query.new('database', 'label_values(pg_stat_database_numbackends{job="$job"}, datname)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)
  + var.query.withSort(1)
  + var.query.generalOptions.withLabel('Database');

// ─── Shared PromQL ───────────────────────────────────────────────────────────

local cacheHitRatio = |||
  sum(rate(pg_stat_database_blks_hit{job="$job", datname="$database"}[$__rate_interval]))
  /
  (
    sum(rate(pg_stat_database_blks_hit{job="$job", datname="$database"}[$__rate_interval]))
    + sum(rate(pg_stat_database_blks_read{job="$job", datname="$database"}[$__rate_interval]))
  )
|||;

// ─── Dashboard ───────────────────────────────────────────────────────────────

g.dashboard.new('PostgreSQL Overview')
+ g.dashboard.withUid('postgres-overview')
+ g.dashboard.withDescription('PostgreSQL metrics from postgres_exporter: connections, transactions, cache, locks.')
+ g.dashboard.time.withFrom('now-1h')
+ g.dashboard.time.withTo('now')
+ g.dashboard.withRefresh('30s')
+ g.dashboard.withVariables([datasourceVar, jobVar, databaseVar])
+ g.dashboard.withPanels([

  // ── Overview stats (y=0) ─────────────────────────────────────────────────

  statPanel('Active Connections', 'short',
    [target('pg_stat_database_numbackends{job="$job", datname="$database"}', 'connections')],
    6, 4, 0, 0),

  statPanel('Database Size', 'bytes',
    [target('pg_stat_database_size_bytes{job="$job", datname="$database"}', 'size')],
    6, 4, 6, 0),

  statPanel('Cache Hit Ratio', 'percentunit',
    [target(cacheHitRatio, 'cache hit ratio')],
    6, 4, 12, 0,
    thresholds=[
      { color: 'red',    value: null },
      { color: 'orange', value: 0.90 },
      { color: 'green',  value: 0.99 },
    ]),

  statPanel('Transactions / s', 'ops',
    [target(
      'sum(rate(pg_stat_database_xact_commit{job="$job", datname="$database"}[$__rate_interval])) + sum(rate(pg_stat_database_xact_rollback{job="$job", datname="$database"}[$__rate_interval]))',
      'tps')],
    6, 4, 18, 0),

  // ── Connections & Transactions (y=4) ─────────────────────────────────────

  tsPanel('Connections', 'short',
    [target('pg_stat_database_numbackends{job="$job", datname="$database"}', 'active')],
    12, 8, 0, 4),

  tsPanel('Transaction Rate', 'ops', [
    target('rate(pg_stat_database_xact_commit{job="$job", datname="$database"}[$__rate_interval])',   'commits/s',   'A'),
    target('rate(pg_stat_database_xact_rollback{job="$job", datname="$database"}[$__rate_interval])', 'rollbacks/s', 'B'),
  ], 12, 8, 12, 4),

  // ── Cache & Temp (y=12) ───────────────────────────────────────────────────

  tsPanel('Cache Hit Ratio', 'percentunit',
    [target(cacheHitRatio, 'cache hit ratio')],
    12, 8, 0, 12),

  tsPanel('Temp Data Written', 'Bps',
    [target(
      'rate(pg_stat_database_temp_bytes{job="$job", datname="$database"}[$__rate_interval])',
      'temp bytes/s')],
    12, 8, 12, 12),

  // ── Contention (y=20) ─────────────────────────────────────────────────────

  tsPanel('Deadlocks', 'short',
    [target(
      'rate(pg_stat_database_deadlocks{job="$job", datname="$database"}[$__rate_interval])',
      'deadlocks/s')],
    12, 8, 0, 20),

  tsPanel('Locks by Mode', 'short',
    [target(
      'sum by (mode) (pg_locks_count{job="$job", datname="$database"})',
      '{{mode}}')],
    12, 8, 12, 20),

])
