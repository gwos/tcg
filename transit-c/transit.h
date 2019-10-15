#ifndef TRANSIT_H
#define TRANSIT_H

#include <stdbool.h>
#include <stdint.h>
#include <time.h>

/* enums start from dumb value to conform upstream (iota + 1)
 * inspired by: https://stackoverflow.com/a/10966395 */
#define GENERATE_ENUM(ENUM) ENUM,
#define GENERATE_STRING(STRING) #STRING,

#define FOREACH_METRIC_KIND(METRIC_KIND) \
  METRIC_KIND(_METRIC_KIND0)             \
  METRIC_KIND(GAUGE)                     \
  METRIC_KIND(DELTA)                     \
  METRIC_KIND(CUMULATIVE)                \
  METRIC_KIND(METRIC_KIND_UNSPECIFIED)

static const char *METRIC_KIND_STRING[] = {
    FOREACH_METRIC_KIND(GENERATE_STRING)};

typedef enum { FOREACH_METRIC_KIND(GENERATE_ENUM) } METRIC_KIND_ENUM;

#define FOREACH_VALUE_TYPE(VALUE_TYPE) \
  VALUE_TYPE(_VALUE_TYPE0)             \
  VALUE_TYPE(IntegerType)              \
  VALUE_TYPE(DoubleType)               \
  VALUE_TYPE(StringType)               \
  VALUE_TYPE(BooleanType)              \
  VALUE_TYPE(TimeType)                 \
  VALUE_TYPE(UnspecifiedType)

static const char *VALUE_TYPE_STRING[] = {FOREACH_VALUE_TYPE(GENERATE_STRING)};

typedef enum { FOREACH_VALUE_TYPE(GENERATE_ENUM) } VALUE_TYPE_ENUM;

// Our usual trick of using enumeration-constant names to generate equivalent
// strings for JSON encoding is useless here, because the supported
// units strings are a subset of The Unified Code for Units of Measure
// (http://unitsofmeasure.org/ucum.html) standard, which defines units names
// using arbitrary characters which may are not necessarily limited to the
// alphanumerics that are valid for enumeration constants.  We must therefore
// define the associated strings manually here, so JSON encoding and decoding
// will have access to these definitions.  It is therefore mandatory to keep
// the list of enumeration constants and the list of strings synchronized as
// the code evolves.

#define FOREACH_UNIT(UNIT) \
  UNIT(_UNIT0)             \
  UNIT(UnitCounter)        \
  UNIT(PercentCPU)

static const char *UNIT_STRING[] = {
    "",            // _UNIT0, no units specified
    "1",           // UnitCounter
    "%{cpu}",      // PercentCPU, as in load measurements
};

typedef enum { FOREACH_UNIT(GENERATE_ENUM) } UNIT_ENUM;

#define FOREACH_COMPUTE_TYPE(COMPUTE_TYPE) \
  COMPUTE_TYPE(_COMPUTE_TYPE0)             \
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
  MONITOR_STATUS(_MONITOR_STATUS0)             \
  MONITOR_STATUS(SERVICE_OK)                   \
  MONITOR_STATUS(SERVICE_WARNING)              \
  MONITOR_STATUS(SERVICE_UNSCHEDULED_CRITICAL) \
  MONITOR_STATUS(SERVICE_PENDING)              \
  MONITOR_STATUS(SERVICE_SCHEDULED_CRITICAL)   \
  MONITOR_STATUS(SERVICE_UNKNOWN)              \
  MONITOR_STATUS(HOST_UP)                      \
  MONITOR_STATUS(HOST_UNSCHEDULED_DOWN)        \
  MONITOR_STATUS(HOST_PENDING)                 \
  MONITOR_STATUS(HOST_SCHEDULED_DOWN)          \
  MONITOR_STATUS(HOST_UNREACHABLE)

static const char *MONITOR_STATUS_STRING[] = {
    FOREACH_MONITOR_STATUS(GENERATE_STRING)};

typedef enum { FOREACH_MONITOR_STATUS(GENERATE_ENUM) } MONITOR_STATUS_ENUM;

#define FOREACH_METRIC_SAMPLE_TYPE(METRIC_SAMPLE_TYPE) \
  METRIC_SAMPLE_TYPE(_METRIC_SAMPLE_TYPE0)             \
  METRIC_SAMPLE_TYPE(Value)                            \
  METRIC_SAMPLE_TYPE(Warning)                          \
  METRIC_SAMPLE_TYPE(Critical)                         \
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
  time_t endTime, startTime;  // go:time.Time
} TimeInterval;

typedef struct {
  VALUE_TYPE_ENUM valueType;
  bool boolValue;
  double doubleValue;
  int64_t integerValue;
  time_t timeValue;  // go:time.Time
  char *stringValue;
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
  METRIC_SAMPLE_TYPE_ENUM sampleType;
  TimeInterval interval;
  TypedValue value;
} MetricSample;

typedef struct {
  size_t count;
  MetricSample *items;
} MetricSampleList;

typedef struct {
  MONITOR_STATUS_ENUM status;
  char *name, *type, *owner, *category, *description, *lastPlugInOutput;
  time_t lastCheckTime;           // go:time.Time
  time_t nextCheckTime;           // go:time.Time
  TypedValuePairList properties;  // go:map[string]TypedValue
} MonitoredResource;

typedef struct {
  size_t count;
  MonitoredResource *items;
} MonitoredResourceList;

typedef struct {
  char *             metricName;
  MetricSampleList * metricSamples; // go:[]*MetricSample
  StringPairList     tags;          // go:map[string]string
  UNIT_ENUM          unit;
} TimeSeries;

typedef struct {
  size_t count;
  TimeSeries *items;
} TimeSeriesList;

typedef struct {
  char *description;
  char *key;
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
  char *appType, *agentID, *traceToken;
  time_t timeStamp;  // go:time.Time
} TracerContext;

typedef struct {
  char *groupName;
  MonitoredResourceList resources;  // go:[]MonitoredResource
} ResourceGroup;

typedef struct {
  size_t count;
  ResourceGroup *items;
} ResourceGroupList;

typedef struct {
  TracerContext         context;
  MonitoredResourceList inventory;  // go:*[]MonitoredResource
  ResourceGroupList     groups;     // go:*[]ResourceGroup
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
