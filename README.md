# sensu-metric-bridge Asset

Supported formats for InfluxDB ingestion, your metrics endpoint must output one of the following formats:

- simple metric, like  
```
seconds_since_last_successful_run 46598.538422381`
```

- metric with fields and only the $relevantPrefix as identifier, like  
```
metrics_myApp{somefield="DB",someotherfield="TransactionsTotal"} 17
-------------
      ^--- $relevantPrefix argument
```

- metric with fields, $relevantPrefix + another constant identifier, like  
```
myApp_return_proc{somefield="files",someotherfield="err"} 0
      -----------
-----      ^--- additional identifier
  ^--- $relevantPrefix argument
```
