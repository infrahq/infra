# Metrics

Infra ships with Prometheus metrics built-in. To access the metrics endpoint, go to `$INFRA_SERVER:9090/metrics`.

## Sample Output

```
$ curl localhost:9090/metrics
# HELP build_info Build information about Infra Server.
# TYPE build_info gauge
build_info{branch="main",commit="",date="",version="0.0.0-development"} 1
# HELP database_connected Database connection status.
# TYPE database_connected gauge
database_connected 1
# HELP database_info Information about configured database.
# TYPE database_info gauge
database_info{name="sqlite"} 1
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 6.3958e-05
go_gc_duration_seconds{quantile="0.25"} 8.3042e-05
go_gc_duration_seconds{quantile="0.5"} 8.8167e-05
go_gc_duration_seconds{quantile="0.75"} 0.000166625
go_gc_duration_seconds{quantile="1"} 0.000167333
go_gc_duration_seconds_sum 0.000655124
go_gc_duration_seconds_count 6
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 30
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.16.13"} 1
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 1.0739064e+07
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 2.5754896e+07
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 1.456602e+06
# HELP go_memstats_frees_total Total number of frees.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total 220329
# HELP go_memstats_gc_cpu_fraction The fraction of this program's available CPU time used by the GC since the program started.
# TYPE go_memstats_gc_cpu_fraction gauge
go_memstats_gc_cpu_fraction 0.0025359370371158844
# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 5.898248e+06
# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and still in use.
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes 1.0739064e+07
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 5.2887552e+07
# HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use.
# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 1.3336576e+07
# HELP go_memstats_heap_objects Number of allocated objects.
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects 82656
# HELP go_memstats_heap_released_bytes Number of heap bytes released to OS.
# TYPE go_memstats_heap_released_bytes gauge
go_memstats_heap_released_bytes 5.1511296e+07
# HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system.
# TYPE go_memstats_heap_sys_bytes gauge
go_memstats_heap_sys_bytes 6.6224128e+07
# HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
# TYPE go_memstats_last_gc_time_seconds gauge
go_memstats_last_gc_time_seconds 1.6419263434352999e+09
# HELP go_memstats_lookups_total Total number of pointer lookups.
# TYPE go_memstats_lookups_total counter
go_memstats_lookups_total 0
# HELP go_memstats_mallocs_total Total number of mallocs.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total 302985
# HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures.
# TYPE go_memstats_mcache_inuse_bytes gauge
go_memstats_mcache_inuse_bytes 4800
# HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system.
# TYPE go_memstats_mcache_sys_bytes gauge
go_memstats_mcache_sys_bytes 16384
# HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures.
# TYPE go_memstats_mspan_inuse_bytes gauge
go_memstats_mspan_inuse_bytes 193664
# HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system.
# TYPE go_memstats_mspan_sys_bytes gauge
go_memstats_mspan_sys_bytes 196608
# HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place.
# TYPE go_memstats_next_gc_bytes gauge
go_memstats_next_gc_bytes 1.24e+07
# HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations.
# TYPE go_memstats_other_sys_bytes gauge
go_memstats_other_sys_bytes 903718
# HELP go_memstats_stack_inuse_bytes Number of bytes in use by the stack allocator.
# TYPE go_memstats_stack_inuse_bytes gauge
go_memstats_stack_inuse_bytes 884736
# HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator.
# TYPE go_memstats_stack_sys_bytes gauge
go_memstats_stack_sys_bytes 884736
# HELP go_memstats_sys_bytes Number of bytes obtained from system.
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes 7.5580424e+07
# HELP go_threads Number of OS threads created.
# TYPE go_threads gauge
go_threads 9
# HELP http_requests_duration_seconds A histogram of the duration, in seconds, handling HTTP requests.
# TYPE http_requests_duration_seconds histogram
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.001"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.002"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.004"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.008"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.016"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.032"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.064"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.128"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.256"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="0.512"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="1.024"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="2.048"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="4.096"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="8.192"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="16.384"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations",method="GET",status="200",le="+Inf"} 1
http_requests_duration_seconds_sum{handler="/v1/destinations",method="GET",status="200"} 0.000855667
http_requests_duration_seconds_count{handler="/v1/destinations",method="GET",status="200"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.001"} 0
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.002"} 0
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.004"} 0
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.008"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.016"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.032"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.064"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.128"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.256"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="0.512"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="1.024"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="2.048"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="4.096"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="8.192"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="16.384"} 1
http_requests_duration_seconds_bucket{handler="/v1/destinations/:id",method="PUT",status="200",le="+Inf"} 1
http_requests_duration_seconds_sum{handler="/v1/destinations/:id",method="PUT",status="200"} 0.006805916
http_requests_duration_seconds_count{handler="/v1/destinations/:id",method="PUT",status="200"} 1
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.001"} 2
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.002"} 6
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.004"} 11
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.008"} 11
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.016"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.032"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.064"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.128"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.256"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="0.512"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="1.024"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="2.048"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="4.096"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="8.192"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="16.384"} 12
http_requests_duration_seconds_bucket{handler="/v1/grants",method="GET",status="200",le="+Inf"} 12
http_requests_duration_seconds_sum{handler="/v1/grants",method="GET",status="200"} 0.031218166
http_requests_duration_seconds_count{handler="/v1/grants",method="GET",status="200"} 12
# HELP http_requests_in_progress Number of HTTP requests currently in progress.
# TYPE http_requests_in_progress gauge
http_requests_in_progress{handler="/v1/destinations",method="GET"} 0
http_requests_in_progress{handler="/v1/destinations/:id",method="PUT"} 0
http_requests_in_progress{handler="/v1/grants",method="GET"} 0
# HELP http_requests_total Total number of HTTP requests served.
# TYPE http_requests_total counter
http_requests_total{handler="/v1/destinations",method="GET",status="200"} 1
http_requests_total{handler="/v1/destinations/:id",method="PUT",status="200"} 1
http_requests_total{handler="/v1/grants",method="GET",status="200"} 12
# HELP infra_destinations Number of destinations managed by Infra.
# TYPE infra_destinations gauge
infra_destinations 1
# HELP infra_grants Number of grants managed by Infra.
# TYPE infra_grants gauge
infra_grants 1
# HELP infra_groups Number of groups managed by Infra.
# TYPE infra_groups gauge
infra_groups 4
# HELP infra_providers Number of providers managed by Infra.
# TYPE infra_providers gauge
infra_providers 1
# HELP infra_users Number of users managed by Infra.
# TYPE infra_users gauge
infra_users 9
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.29
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1.048576e+06
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 28
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 3.8100992e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.64192634179e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 1.308618752e+09
# HELP process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
# TYPE process_virtual_memory_max_bytes gauge
process_virtual_memory_max_bytes 1.8446744073709552e+19
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 1
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
```
