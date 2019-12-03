package org.groundwork.tng.transit;

import org.groundwork.rs.transit.*;
import org.junit.Test;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;

/**
 * Unit test for simple App.
 */
public class AppTest {

    private static final String TEST_HOST_NAME = "GW8_TNG_TEST_HOST";
    private static final String TEST_SERVICE_NAME = "GW8_TNG_TEST_SERVICE";
    private static final String HOST_RESOURCE_TYPE = "HOST";
    private static final String SERVICE_RESOURCE_TYPE = "SERVICE";

    /**
     * Test sending resource and metrics
     * <p>
     * Usage:
     * 1. Start GroundWork Foundation server
     * 2. Generate '.so' file by running `go build -o libtransit/libtransit.so -buildmode=c-shared libtransit/libtransit.go` command in TNG
     * 3. Set path to '.so' file in TransitServiceImpl constructor.
     * 4. Run test.
     */
    @Test
    public void shouldSendResourceAndMetrics() throws ParseException {
        TransitServices transit = new TransitServicesImpl();

        DtoTracerContext context = DtoTracerContext.builder()
                .setAgentId("3939333393342")
                .setAppType("VEMA")
                .setTimeStamp(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                .setTraceToken("token-99e93")
                .build();

        DtoTimeSeries timeSeries = DtoTimeSeries.builder()
                .setMetricName(TEST_SERVICE_NAME)
                .setSampleType(DtoMetricSampleType.Value)
                .setInterval(DtoTimeInterval.builder()
                        .setStartTime(new SimpleDateFormat("dd/MM/yyyy").parse("21/10/2019"))
                        .setEndTime(new SimpleDateFormat("dd/MM/yyyy").parse("23/10/2019"))
                        .build())
                .setValue(DtoTypedValue.builder()
                        .setValueType(DtoValueType.IntegerType)
                        .setIntegerValue(1)
                        .build())
                .build();

        DtoMonitoredResource resource = DtoMonitoredResource.builder()
                .setName(TEST_HOST_NAME)
                .setType(HOST_RESOURCE_TYPE)
                .setStatus(DtoMonitorStatus.HOST_UP)
                .setLastCheckTime(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                .build();

        DtoMonitoredService service = DtoMonitoredService.builder()
                .setName(TEST_SERVICE_NAME)
                .setType(SERVICE_RESOURCE_TYPE)
                .setStatus(DtoMonitorStatus.SERVICE_OK)
                .setOwner(TEST_HOST_NAME)
                .setLastCheckTime(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                .build();

        service.addMetric(timeSeries);

        resource.addService(service);

        DtoResourcesWithServices resources = DtoResourcesWithServices.builder()
                .setContext(context)
                .build();

        resources.add(resource);

        transit.SendResourcesWithMetrics(resources);
    }


    /**
     * Test synchronize inventory
     * <p>
     * Usage:
     * 1. Start GroundWork Foundation server
     * 2. Generate '.so' file by running `go build -o libtransit/libtransit.so -buildmode=c-shared libtransit/libtransit.go` command in TNG
     * 3. Set path to '.so' file in TransitServiceImpl constructor.
     * 4. Run test.
     */
    @Test
    public void shouldSynchronizeInventory() {
        TransitServices transit = new TransitServicesImpl();

        DtoTracerContext context = DtoTracerContext.builder()
                .setAgentId("3939333393342")
                .setAppType("VEMA")
                .setTimeStamp(new Date())
                .setTraceToken("token-99e93")
                .build();

        DtoInventoryResource host = DtoInventoryResource.builder()
                .setName(TEST_HOST_NAME)
                .setType(HOST_RESOURCE_TYPE)
                .build();
        host.addService(DtoInventoryService.builder()
                .setName(TEST_SERVICE_NAME)
                .setType("network-device")
                .setOwner(TEST_HOST_NAME)
                .build());

        DtoInventory dtoInventory = new DtoInventory();
        dtoInventory.setContext(context);
        dtoInventory.add(host);

        transit.SynchronizeInventory(dtoInventory);
    }

    private static final Integer TEST_SERVICES_COUNT = 10;
    private static final Integer TEST_METRICS_COUNT = 5;

    /**
     * Test synchronize inventory
     * <p>
     * Usage:
     * 1. Start GroundWork Foundation server
     * 2. Generate '.so' file by running `go build -o libtransit/libtransit.so -buildmode=c-shared libtransit/libtransit.go` command in TNG
     * 3. Set path to '.so' file in TransitServiceImpl constructor.
     * 4. Set preferred TEST_SERVICES_COUNT and TEST_METRICS_COUNT constants.
     * 5. Run test.
     */
    @Test
    public void testSynchronizeInventoryPerformance() throws IOException {
        TransitServices transit = new TransitServicesImpl();

        DtoTracerContext context = DtoTracerContext.builder()
                .setAgentId("3939333393342")
                .setAppType("VEMA")
                .setTimeStamp(new Date())
                .setTraceToken("token-99e93")
                .build();

        DtoInventoryResource host = DtoInventoryResource.builder()
                .setName(TEST_HOST_NAME)
                .setType(HOST_RESOURCE_TYPE)
                .build();

        DtoGroup group = new DtoGroup();
        group.setGroupName("GW8");
        group.setResources(new ArrayList<>());

        DtoInventory dtoInventory = new DtoInventory();
        dtoInventory.setContext(context);
        dtoInventory.add(host);
        dtoInventory.add(group);

        for (int i = 0; i < TEST_SERVICES_COUNT; i++) {
            dtoInventory.add(DtoInventoryResource.builder()
                    .setName(TEST_SERVICE_NAME + "_" + i)
                    .setType(SERVICE_RESOURCE_TYPE)
                    .setOwner(TEST_HOST_NAME)
                    .build());
            transit.SynchronizeInventory(dtoInventory);
        }

        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
        String name = reader.readLine();
    }

    /**
     * Test synchronize inventory
     * <p>
     * Usage:
     * 1. Start GroundWork Foundation server
     * 2. Generate '.so' file by running `go build -o libtransit/libtransit.so -buildmode=c-shared libtransit/libtransit.go` command in TNG
     * 3. Set path to '.so' file in TransitServiceImpl constructor.
     * 4. Set preferred TEST_SERVICES_COUNT and TEST_METRICS_COUNT constants.
     * 5. Run test.
     */
    @Test
    public void testSendResourceWithMetricsPerformance() throws ParseException, IOException {
        TransitServices transit = new TransitServicesImpl();

        DtoTracerContext context = DtoTracerContext.builder()
                .setAgentId("3939333393342")
                .setAppType("VEMA")
                .setTimeStamp(new Date())
                .setTraceToken("token-99e93")
                .build();

        DtoTimeSeries timeSeries = DtoTimeSeries.builder()
                .setMetricName(TEST_SERVICE_NAME)
                .setSampleType(DtoMetricSampleType.Warning)
                .setInterval(DtoTimeInterval.builder()
                        .setStartTime(new SimpleDateFormat("dd/MM/yyyy").parse("21/10/2019"))
                        .setEndTime(new SimpleDateFormat("dd/MM/yyyy").parse("23/10/2019"))
                        .build())
                .setValue(DtoTypedValue.builder()
                        .setValueType(DtoValueType.IntegerType)
                        .setIntegerValue(1)
                        .build())
                .build();

        DtoResourcesWithServices resources = DtoResourcesWithServices.builder().setContext(context).build();

        DtoMonitoredResource host = DtoMonitoredResource.builder()
                .setName(TEST_HOST_NAME)
                .setType(HOST_RESOURCE_TYPE)
                .setStatus(DtoMonitorStatus.HOST_UP)
                .setLastCheckTime(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                .build();

        resources.add(host);

        for (int i = 0; i < TEST_SERVICES_COUNT; i++) {
            for (int j = 0; j < TEST_METRICS_COUNT; j++) {
                DtoMonitoredService service = DtoMonitoredService.builder()
                        .setName(TEST_SERVICE_NAME + "_" + i)
                        .setType(SERVICE_RESOURCE_TYPE)
                        .setLastCheckTime(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                        .setOwner(TEST_HOST_NAME)
                        .build();

                if (i % 2 == 0) {
                    service.setStatus(DtoMonitorStatus.SERVICE_OK);
                } else {
                    service.setStatus(DtoMonitorStatus.SERVICE_WARNING);
                }

                service.addMetric(timeSeries);
                host.addService(service);

                transit.SendResourcesWithMetrics(resources);
            }
        }

        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
        String name = reader.readLine();
    }

    /*Example of callback func*/
    static ListMetricsCallback func = new ListMetricsCallback() {
        @Override
        public String GetTextHandlerType() {
            //Metric list generation ...

            return "List metric list";
        }
    };
}
