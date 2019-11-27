The JSON strings presented here were used as the initial basis for the unit
tests for the gotocjson tool.  In some cases, the JSON strings shown here are
subtly wrong, and corrections have been made in the gotocjson/unittest.c code.

>TimeInterval:  

    {"endTime":1572955806397,"startTime":1572955806397}

>TypedValue:  

    {"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}

>MetricSample:  

    {"sampleType":"Warning","interval":{"endTime":1572955806397,"startTime":1572955806397},"value":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}

>TimeSeries:  

    {"metricName":"TestMetric","metricSamples":[{"sampleType":"Warning","interval":{"endTime":1572955806397,"startTime":1572955806397},"value":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}],"tags":{"tag1":"TAG_1","tag2":"TAG_2"},"unit":"1"}

>MetricDescriptor:  

    {"name":"TestCustomName","description":"TestDescription","displayName":"TestDisplayName","labels":[{"description":"TestDescription","key":"TestKey","valueType":"TestValue"}],"thresholds":[{"key":"TestKey","value":2}],"type":"TestType","unit":"1","valueType":"IntegerType","computeType":"Query","metricKind":"GAUGE"}

>LabelDescriptor:  

    {"description":"TestDescription","key":"TestKey","valueType":"TestValue"}

>ThresholdDescriptor:  
    
    {"key":"TestKey","value":2}

>InventoryResource:  

    {"name":"TestName","type":"TestType","owner":"TestOwner","category":"TestCategory","description":"TestDescription","device":"TestDevice","properties":{"property":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}}

>ResourceStatus:  

    {"name":"TestName","type":"TestType","owner":"TestOwner","status":"SERVICE_OK","lastCheckTime":1572955806398,"nextCheckTime":1572955806398,"lastPlugInOutput":"Test plugin output","properties":{"property":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}}

>MonitoredResource:  

    {"name":"TestName","type":"TestType","owner":"TestOwner"}

>TracerContext:  

    {"appType":"TestAppType","agentID":"TestAgentID","traceToken":"TestTraceToken","timeStamp":1572955806398}

>SendInventoryRequest:  

    {"resources":[{"name":"TestName","type":"TestType","owner":"TestOwner","category":"TestCategory","description":"TestDescription","device":"TestDevice","properties":{"property":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}}],"groups":[{"groupName":"TestGroupName","resources":[{"name":"TestName","type":"TestType","owner":"TestOwner"}]}]}

>OperationResult:  

    {"entity":"TestEntity","status":"TestStatus","message":"TestMessage","location":"TestLocation","entityID":173}

>OperationResults:  

    {"successful":1,"failed":0,"entityType":"TestEntity","operation":"Insert","warning":0,"count":1,"results":[{"entity":"TestEntity","status":"TestStatus","message":"TestMessage","location":"TestLocation","entityID":173}]}

>ResourceGroup:  

    {"groupName":"TestGroupName","resources":[{"name":"TestName","type":"TestType","owner":"TestOwner"}]}

>ResourceWithMetrics:  

    {"resource":{"name":"TestName","type":"TestType","owner":"TestOwner","status":"SERVICE_OK","lastCheckTime":1572955806398,"nextCheckTime":1572955806398,"lastPlugInOutput":"Test plugin output","properties":{"property":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}},"metrics":[{"metricName":"TestMetric","metricSamples":[{"sampleType":"Warning","interval":{"endTime":1572955806397,"startTime":1572955806397},"value":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}],"tags":{"tag1":"TAG_1","tag2":"TAG_2"},"unit":"1"}]}

>ResourceWithMetricsRequest:  

    {"context":{"appType":"TestAppType","agentID":"TestAgentID","traceToken":"TestTraceToken","timeStamp":1572955806398},"resources":[{"resource":{"name":"TestName","type":"TestType","owner":"TestOwner","status":"SERVICE_OK","lastCheckTime":1572955806398,"nextCheckTime":1572955806398,"lastPlugInOutput":"Test plugin output","properties":{"property":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}},"metrics":[{"metricName":"TestMetric","metricSamples":[{"sampleType":"Warning","interval":{"endTime":1572955806397,"startTime":1572955806397},"value":{"valueType":"IntegerType","integerValue":1,"timeValue":1572955806397}}],"tags":{"tag1":"TAG_1","tag2":"TAG_2"},"unit":"1"}]}]}

>AgentStats

    {"AgentID":"TestAgentId","AppType":"TestAgentType","BytesSent":1567,"MetricsSent":12,"MessagesSent":4,"LastInventoryRun":1572958409541,"LastMetricsRun":1572958409541,"ExecutionTimeInventory":133,"ExecutionTimeMetrics":234,"UpSince":1572958409541,"LastError":"Test last error"}
