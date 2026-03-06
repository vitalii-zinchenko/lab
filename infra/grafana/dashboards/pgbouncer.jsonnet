// PgBouncer Overview dashboard.
// Metrics sourced from prometheus-community/pgbouncer_exporter v0.11+ (port 9127).
// Covers client/server connections, pool saturation, throughput, and wait time.

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
  var.query.new('job', 'label_values(pgbouncer_up, job)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)
  + var.query.withSort(1)
  + var.query.generalOptions.withLabel('Job');

local databaseVar =
  var.query.new('database', 'label_values(pgbouncer_pools_client_active_connections{job="$job"}, database)')
  + var.query.withDatasource('prometheus', '${datasource}')
  + var.query.withRefresh(2)
  + var.query.withSort(1)
  + var.query.generalOptions.withLabel('Database');

// ─── Dashboard ───────────────────────────────────────────────────────────────

g.dashboard.new('PgBouncer Overview')
+ g.dashboard.withUid('pgbouncer-overview')
+ g.dashboard.withDescription('PgBouncer metrics from pgbouncer_exporter: pool connections, wait time, throughput.')
+ g.dashboard.time.withFrom('now-1h')
+ g.dashboard.time.withTo('now')
+ g.dashboard.withRefresh('30s')
+ g.dashboard.withVariables([datasourceVar, jobVar, databaseVar])
+ g.dashboard.withPanels([

  // ── Overview stats (y=0) ─────────────────────────────────────────────────

  statPanel('Active Clients', 'short',
    [target(
      'sum(pgbouncer_pools_client_active_connections{job="$job", database="$database"})',
      'active clients')],
    6, 4, 0, 0),

  // Alert-worthy: clients waiting for a free server connection.
  statPanel('Waiting Clients', 'short',
    [target(
      'sum(pgbouncer_pools_client_waiting_connections{job="$job", database="$database"})',
      'waiting clients')],
    6, 4, 6, 0,
    thresholds=[
      { color: 'green',  value: null },
      { color: 'orange', value: 1    },
      { color: 'red',    value: 10   },
    ]),

  statPanel('Active Servers', 'short',
    [target(
      'sum(pgbouncer_pools_server_active_connections{job="$job", database="$database"})',
      'active servers')],
    6, 4, 12, 0),

  statPanel('Idle Servers', 'short',
    [target(
      'sum(pgbouncer_pools_server_idle_connections{job="$job", database="$database"})',
      'idle servers')],
    6, 4, 18, 0),

  // ── Connection Pool (y=4) ─────────────────────────────────────────────────

  tsPanel('Client Connections', 'short', [
    target('sum(pgbouncer_pools_client_active_connections{job="$job", database="$database"})',  'active',  'A'),
    target('sum(pgbouncer_pools_client_waiting_connections{job="$job", database="$database"})', 'waiting', 'B'),
  ], 12, 8, 0, 4),

  tsPanel('Server Connections', 'short', [
    target('sum(pgbouncer_pools_server_active_connections{job="$job", database="$database"})', 'active', 'A'),
    target('sum(pgbouncer_pools_server_idle_connections{job="$job", database="$database"})',   'idle',   'B'),
    target('sum(pgbouncer_pools_server_used_connections{job="$job", database="$database"})',   'used',   'C'),
  ], 12, 8, 12, 4),

  // ── Throughput (y=12) ─────────────────────────────────────────────────────

  tsPanel('Transaction Rate', 'ops',
    [target(
      'rate(pgbouncer_stats_totals_sql_transactions_pooled_total{job="$job", database="$database"}[$__rate_interval])',
      'xact/s')],
    12, 8, 0, 12),

  tsPanel('Network Traffic', 'Bps', [
    target('rate(pgbouncer_stats_totals_received_bytes_total{job="$job", database="$database"}[$__rate_interval])', 'received/s', 'A'),
    target('rate(pgbouncer_stats_totals_sent_bytes_total{job="$job", database="$database"}[$__rate_interval])',     'sent/s',     'B'),
  ], 12, 8, 12, 12),

  // ── Latency (y=20) ────────────────────────────────────────────────────────

  // Max wait: how long the slowest client has been waiting for a server conn.
  tsPanel('Max Client Wait Time', 's',
    [target(
      'max(pgbouncer_pools_client_maxwait_seconds{job="$job", database="$database"})',
      'max wait')],
    12, 8, 0, 20),

  // Average query round-trip as seen by pgbouncer (in seconds).
  tsPanel('Avg Query Duration', 's',
    [target(
      |||
        rate(pgbouncer_stats_totals_queries_duration_seconds_total{job="$job", database="$database"}[$__rate_interval])
        /
        rate(pgbouncer_stats_totals_queries_pooled_total{job="$job", database="$database"}[$__rate_interval])
      |||,
      'avg query duration')],
    12, 8, 12, 20),

])
