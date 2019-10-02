#ifndef TRANSIT_H
#define TRANSIT_H

#include <stdbool.h>
#include <stdint.h>

/* https://stackoverflow.com/a/10966395 */
#define GENERATE_ENUM(ENUM) ENUM,
#define GENERATE_STRING(STRING) #STRING,

#define FOREACH_METRIC_KIND(METRIC_KIND) \
  METRIC_KIND(GAUGE)                     \
  METRIC_KIND(DELTA)                     \
  METRIC_KIND(CUMULATIVE)                \
  METRIC_KIND(METRIC_KIND_UNSPECIFIED)

static const char *METRIC_KIND_STRING[] = {
    FOREACH_METRIC_KIND(GENERATE_STRING)};

typedef enum { FOREACH_METRIC_KIND(GENERATE_ENUM) } METRIC_KIND_ENUM;

#define FOREACH_VALUE_TYPE(VALUE_TYPE) \
  VALUE_TYPE(IntegerType)              \
  VALUE_TYPE(DoubleType)               \
  VALUE_TYPE(StringType)               \
  VALUE_TYPE(BooleanType)              \
  VALUE_TYPE(DateType)                 \
  VALUE_TYPE(UnspecifiedType)

static const char *VALUE_TYPE_STRING[] = {FOREACH_VALUE_TYPE(GENERATE_STRING)};

typedef enum { FOREACH_VALUE_TYPE(GENERATE_ENUM) } VALUE_TYPE_ENUM;

#define FOREACH_UNIT(UNIT) UNIT(UnitCounter)

static const char *UNIT_STRING[] = {FOREACH_UNIT(GENERATE_STRING)};

typedef enum { FOREACH_UNIT(GENERATE_ENUM) } UNIT_ENUM;

#define FOREACH_COMPUTE_TYPE(COMPUTE_TYPE) \
  COMPUTE_TYPE(query)                      \
  COMPUTE_TYPE(regex)                      \
  COMPUTE_TYPE(synthetic)                  \
  COMPUTE_TYPE(info)                       \
  COMPUTE_TYPE(performance)                \
  COMPUTE_TYPE(health)

static const char *COMPUTE_TYPE_STRING[] = {
    FOREACH_COMPUTE_TYPE(GENERATE_STRING)};

typedef enum { FOREACH_COMPUTE_TYPE(GENERATE_ENUM) } COMPUTE_TYPE_ENUM;

#define FOREACH_MONITOR_STATUS(MONITOR_STATUS) \
  MONITOR_STATUS(SERVICE_OK)                   \
  MONITOR_STATUS(SERVICE_UNSCHEDULED_CRITICAL) \
  MONITOR_STATUS(SERVICE_WARNING)              \
  MONITOR_STATUS(SERVICE_PENDING)              \
  MONITOR_STATUS(SERVICE_SCHEDULED_CRITICAL)   \
  MONITOR_STATUS(SERVICE_UNKNOWN)              \
  MONITOR_STATUS(HOST_UP)                      \
  MONITOR_STATUS(HOST_UNSCHEDULED_DOWN)        \
  MONITOR_STATUS(HOST_WARNING)                 \
  MONITOR_STATUS(HOST_PENDING)                 \
  MONITOR_STATUS(HOST_SCHEDULED_DOWN)          \
  MONITOR_STATUS(HOST_UNREACHABLE)

static const char *MONITOR_STATUS_STRING[] = {
    FOREACH_MONITOR_STATUS(GENERATE_STRING)};

typedef enum { FOREACH_MONITOR_STATUS(GENERATE_ENUM) } MONITOR_STATUS_ENUM;

#define FOREACH_METRIC_SAMPLE_TYPE(METRIC_SAMPLE_TYPE) \
  METRIC_SAMPLE_TYPE(Value)                            \
  METRIC_SAMPLE_TYPE(Critical)                         \
  METRIC_SAMPLE_TYPE(Warning)                          \
  METRIC_SAMPLE_TYPE(Min)                              \
  METRIC_SAMPLE_TYPE(Max)

static const char *METRIC_SAMPLE_TYPE_STRING[] = {
    FOREACH_METRIC_SAMPLE_TYPE(GENERATE_STRING)};

typedef enum {
  FOREACH_METRIC_SAMPLE_TYPE(GENERATE_ENUM)
} METRIC_SAMPLE_TYPE_ENUM;

static const char *SERVICE_RESOURCE = "service";
static const char *HOST_RESOURCE = "host";

typedef struct {
  char *key, *value;
} StringPair;

typedef struct {
  size_t count;
  StringPair *items;
} StringPairList;

typedef struct {
  char *endTime, *startTime;  // go:time.Time
} TimeInterval;

typedef struct {
  VALUE_TYPE_ENUM valueType;
  bool boolValue;
  double doubleValue;
  int64_t integerValue;
  char *stringValue;
  char *dateValue;  // go:time.Time
} TypedValue;

typedef struct {
  char *key;
  TypedValue value;
} TypedValuePair;

typedef struct {
  size_t count;
  TypedValuePair *items;
} TypedValuePairList;

typedef struct {
  TimeInterval interval;
  TypedValue value;
} Point;

typedef struct {
  MONITOR_STATUS_ENUM status;
  char *name, *type, *owner;
  char *lastCheckTime;  // go:time.Time
  char *nextCheckTime;  // go:time.Time
  char *lastPlugInOutput, *category, *description;
  TypedValuePairList properties;  // go:map[string]TypedValue
} MonitoredResource;

typedef struct {
  size_t count;
  MonitoredResource *items;
} MonitoredResourceList;

typedef struct {
  char *metricName;
  METRIC_SAMPLE_TYPE_ENUM sampleType;
  StringPairList tags;  // go:map[string]string
  TimeInterval interval;
  TypedValue value;
  UNIT_ENUM unit;
} TimeSeries;

typedef struct {
  size_t count;
  TimeSeries *items;
} TimeSeriesList;

typedef struct {
  char *description, *key;
  VALUE_TYPE_ENUM valueType;
} LabelDescriptor;

typedef struct {
  size_t count;
  LabelDescriptor *items;
} LabelDescriptorList;

typedef struct {
  char *key;
  int32_t value;
} ThresholdDescriptor;

typedef struct {
  size_t count;
  ThresholdDescriptor *items;
} ThresholdDescriptorList;

typedef struct {
  char *name, *description, *displayName, *type;
  LabelDescriptorList *labels;          // go:[]*LabelDescriptor
  ThresholdDescriptorList *thresholds;  // go:[]*ThresholdDescriptor
  UNIT_ENUM unit;
  VALUE_TYPE_ENUM valueType;
  COMPUTE_TYPE_ENUM computeType;
  METRIC_KIND_ENUM metricKind;
} MetricDescriptor;

typedef struct {
  char *appType, *agentId, *traceToken;
  char *timeStamp;  // go:time.Time
} TracerContext;

typedef struct {
  char *groupName;
  MonitoredResourceList resources;  // go:[]MonitoredResource
} Group;

typedef struct {
  size_t count;
  Group *items;
} GroupList;

typedef struct {
  TracerContext context;
  MonitoredResourceList inventory;  // go:*[]MonitoredResource
  GroupList groups;                 // go:*[]Group
} TransitSendInventoryRequest;

typedef struct {
  int resourcesAdded, resourcesDeleted;
} TransitSynchronizeResponse;

typedef struct {
  MonitoredResource resource;
  TimeSeriesList metrics;  // go:[]TimeSeries
} ResourceWithMetrics;

typedef struct {
  size_t count;
  ResourceWithMetrics *items;
} ResourceWithMetricsList;

typedef struct {
  TracerContext context;
  ResourceWithMetricsList resources;  // go:[]ResourceWithMetrics
} ResourceWithMetricsRequest;

typedef struct {
  char *hostName, *account, *token;
  bool ssl;
} GroundworkConfig;

typedef struct {
  char *user, *password;
} Credentials;

typedef struct {
  GroundworkConfig config;
} Transit;

#endif /* TRANSIT_H */
