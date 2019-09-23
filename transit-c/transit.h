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
  VALUE_TYPE(BOOL)                     \
  VALUE_TYPE(INT8)                     \
  VALUE_TYPE(INT32)                    \
  VALUE_TYPE(INT64)                    \
  VALUE_TYPE(DOUBLE)                   \
  VALUE_TYPE(STRING)                   \
  VALUE_TYPE(VALUE_TYPE_UNSPECIFIED)

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

static const char *SERVICE_RESOURCE = "service";
static const char *HOST_RESOURCE = "host";

typedef struct {
  char *endTime, *startTime;  // time.Time
} TimeInterval;

typedef struct {
  bool boolValue;
  double doubleValue;
  int8_t int8Value;
  int32_t int32Value;
  int64_t int64Value;
  char *stringValue;
} TypedValue;

typedef struct {
  TimeInterval interval;
  TypedValue value;
} Point;

typedef struct {
  char *type;
  TypedValue *labels;  // map[string]TypedValue
} Metric;

typedef struct {
  char *key, *value;
} StringPair;

typedef struct {
  size_t count;
  StringPair *items;
} StringPairs;

typedef struct MonitoredResource {
  char *name, *type;
  MONITOR_STATUS_ENUM status;
  struct MonitoredResource *owner;
  StringPairs labels;  // map[string]string
} MonitoredResource;

typedef struct {
  Metric metric;
  METRIC_KIND_ENUM metricKind;
  Point *points;
  MonitoredResource resource;
  VALUE_TYPE_ENUM valueType;
} TimeSeries;

typedef struct {
  char *description, *key;
  VALUE_TYPE_ENUM valueType;
} LabelDescriptor;

typedef struct {
  char *key;
  int32_t value;
} ThresholdDescriptor;

typedef struct {
  COMPUTE_TYPE_ENUM computeType;
  char *name, *description, *displayName;
  char *labels;
  METRIC_KIND_ENUM metricKind;
  ThresholdDescriptor *thresholds;
  UNIT_ENUM unit;
  VALUE_TYPE_ENUM valueType;
} MetricDescriptor;

typedef struct {
  char *type, *description, *displayName;
  LabelDescriptor *labels;
} MonitoredResourceDescriptor;

typedef struct {
  char *appType, *agentId, *traceToken;
  char *timeStamp;  // time.Time
} TracerContext;

typedef struct {
  char *groupName;
  MonitoredResource *resources;
} Group;

typedef struct {
  TracerContext context;
  MonitoredResource *inventory;
  Group *groups;
} TransitSendInventoryRequest;

typedef struct {
  int resourcesAdded, resourcesDeleted;
} TransitSynchronizeResponse;

typedef struct {
  char *hostName, *account, *token;
  bool ssl;
} GroundworkConfig;

typedef struct {
  GroundworkConfig config;
} Transit;

#endif /* TRANSIT_H */
