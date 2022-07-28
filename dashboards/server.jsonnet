local grafana = import 'grafonnet/grafana.libsonnet';
local prometheus = grafana.prometheus;
local template = grafana.template;
local dashboard = grafana.dashboard;
local transformation = grafana.transformation;

local grafana7 = import 'grafonnet-7.0/grafana.libsonnet';
local stat = grafana7.panel.stat;
local table = grafana7.panel.table;
local graph = grafana7.panel.graph;

local astimeseries(x, format='short', stack=false) = x {
  type: 'timeseries',
  options: {
    tooltip: {
      mode: 'multi',
    },
  },
  fieldConfig: {
    defaults: {
      unit: format,
      custom: {
        stacking: {
          mode: if stack then 'normal' else 'none',
        },
      },
    },
  },
};

dashboard.new(
  'Infra Server',
  schemaVersion=16,
  timezone='utc',
  uid='512571c8648d41f68e7f595261d51a71',
)
.addTemplates([
  template.datasource(
    'datasource',
    'prometheus',
    'Prometheus',
  ),
  template.new(
    'job',
    '$datasource',
    'label_values(build_info{container="server"}, job)',
    label='Job',
    refresh='time',
  ),
  template.new(
    'instance',
    '$datasource',
    'label_values(build_info{container="server", job="$job"}, instance)',
    label='Instance',
    refresh='time',
    includeAll=true,
    multi=true,
  ),
  template.new(
    'http_request_path',
    '$datasource',
    'label_values(http_request_duration_seconds_count{job="$job", instance=~"$instance"}, path)',
    label='HTTP Request',
    refresh='time',
  ),
  template.new(
    'http_request_method',
    '$datasource',
    'label_values(http_request_duration_seconds_count{job="$job", instance=~"$instance", path="$http_request_path"}, method)',
    label='HTTP Method',
    refresh='time',
    includeAll=true,
    multi=true,
  ),
])
.addPanels([
  stat.new(datasource='$datasource')
    .setGridPos(x=0, y=0, h=6, w=24)
    .addTarget(prometheus.target(
      'sum(build_info{job="$job"})',
      legendFormat='Num. of Replicas',
    ))
    .addTarget(prometheus.target(
      'sum by (db_name) (go_sql_open_connections{job="$job"})',
      legendFormat='Num. of Open {{ db_name }} Connections',
    ))
    .addTarget(prometheus.target(
      'sum(infra_providers{job="$job"}) / sum(build_info{job="$job"})',
      legendFormat='Num. of Providers',
    ))
    .addTarget(prometheus.target(
      'sum(infra_users{job="$job"}) / sum(build_info{job="$job"})',
      legendFormat='Num. of Users',
    ))
    .addTarget(prometheus.target(
      'sum(infra_groups{job="$job"}) / sum(build_info{job="$job"})',
      legendFormat='Num. of Groups',
    ))
    .addTarget(prometheus.target(
      'sum(infra_destinations{job="$job"}) / sum(build_info{job="$job"})',
      legendFormat='Num. of Destinations',
    ))
    .addTarget(prometheus.target(
      'sum(infra_grants{job="$job"}) / sum(build_info{job="$job"})',
      legendFormat='Num. of Grants',
    )),
  astimeseries(graph.new(title='HTTP Request Latency', datasource='$datasource')
    .setGridPos(x=0, y=6, h=10, w=12)
    .addTarget(prometheus.target(|||
      sum by (status) (rate(http_request_duration_seconds_sum{job="$job", instance=~"$instance"}[$__rate_interval])) /
      sum by (status) (rate(http_request_duration_seconds_count{job="$job", instance=~"$instance"}[$__rate_interval]))
    |||, legendFormat='{{ status }} (Mean)'))
    .addTarget(prometheus.target(
      'histogram_quantile(0.95, sum by (le, status) (rate(http_request_duration_seconds_bucket{job="$job", instance=~"$instance"}[$__rate_interval])))',
      legendFormat='{{ status }} (95th)')), format='s'),
  astimeseries(graph.new(title='HTTP Error Rate', datasource='$datasource')
    .setGridPos(x=12, y=6, h=10, w=12)
    .addTarget(prometheus.target(|||
      sum by (status) (rate(http_request_duration_seconds_count{job="$job", instance=~"$instance", status=~"4..|5.."}[$__rate_interval])) /
      sum by (status) (rate(http_request_duration_seconds_count{job="$job", instance=~"$instance"}[$__rate_interval]))
    |||, legendFormat='{{ status }}')), format='percentunit'),
  astimeseries(graph.new(title='HTTP Request Latency - $http_request_path', datasource='$datasource')
    .setGridPos(x=0, y=6, h=10, w=12)
    .addTarget(prometheus.target(|||
      sum by (method, status) (rate(http_request_duration_seconds_sum{job="$job", instance=~"$instance", path="$http_request_path", method=~"$http_request_method"}[$__rate_interval])) /
      sum by (method, status) (rate(http_request_duration_seconds_count{job="$job", instance=~"$instance", path="$http_request_path", method=~"$http_request_method"}[$__rate_interval]))
    |||, legendFormat='{{ method }} {{ status }} (Mean)'))
    .addTarget(prometheus.target(
      'histogram_quantile(0.95, sum by (le, method, status) (rate(http_request_duration_seconds_bucket{job="$job", instance=~"$instance", path="$http_request_path", method=~"$http_request_method"}[$__rate_interval])))',
      legendFormat='{{ method }} {{ status }} (95th)')), format='s'),
  astimeseries(graph.new(title='HTTP Error Rate - $http_request_path', datasource='$datasource')
    .setGridPos(x=12, y=6, h=10, w=12)
    .addTarget(prometheus.target(|||
      sum by (method, status) (rate(http_request_duration_seconds_count{job="$job", instance=~"$instance", path="$http_request_path", method=~"$http_request_method", status=~"4..|5.."}[$__rate_interval])) /
      sum by (method, status) (rate(http_request_duration_seconds_count{job="$job", instance=~"$instance", path="$http_request_path", method=~"$http_request_method"}[$__rate_interval]))
    |||, legendFormat='{{ method }} {{ status }}')), format='percentunit'),
  astimeseries(graph.new(title='Providers By Kind', datasource='$datasource')
    .setGridPos(x=0, y=26, h=10, w=8)
    .addTarget(prometheus.target(|||
      sum by (kind) (infra_providers{job="$job"}) / ignoring (kind) group_left sum(build_info{job="$job"})
    |||, legendFormat='{{ kind }}')), stack=true),
  astimeseries(graph.new(title='Destinations By Status', datasource='$datasource')
    .setGridPos(x=8, y=26, h=10, w=8)
    .addTarget(prometheus.target(|||
      sum by (status) (infra_destinations{job="$job"}) / ignoring (status) group_left sum(build_info{job="$job"})
    |||, legendFormat='{{ status }}')), stack=true),
  table.new(title='Build Info', datasource='$datasource')
    .setGridPos(x=16, y=26, h=10, w=8)
    .addTarget(prometheus.target(
      'build_info{job="$job"}',
      instant=true,
    )) {
      transformations: [
        transformation.new(
          'labelsToFields',
        ),
        transformation.new(
          'organize',
          options = {
            excludeByName: {
              Time: true,
              Value: true,
              __name__: true,
              container: true
            },
            indexByName: {
              job: 0,
              instance: 1,
              version: 2,
              branch: 3,
              endpoint: 4,
              namespace: 5,
              pod: 6,
              service: 7
            },
          }
        )
      ]
    },
])
