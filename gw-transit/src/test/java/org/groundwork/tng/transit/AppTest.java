package org.groundwork.tng.transit;

import org.groundwork.rs.transit.*;
import org.junit.Test;

import java.io.IOException;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

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
     * 4. Set path to '.so' file in TransitServiceImpl constructor.
     * 3. Run test
     */
    @Test
    public void shouldSendResourceAndMetrics() throws IOException, ParseException {
        TransitServices transit = new TransitServicesImpl();

        DtoTracerContext context = DtoTracerContext.builder()
                .setAgentId("3939333393342")
                .setAppType("VEMA")
                .setTimeStamp(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                .setTraceToken("token-99e93")
                .build();

        List<DtoTimeSeries> timeSeries = new ArrayList<>();
        timeSeries.add(DtoTimeSeries.builder()
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
                .build());

        DtoResourceWithMetricsList resources = DtoResourceWithMetricsList.builder().setContext(context).build();

        resources.add(DtoResourceWithMetrics.builder()
                .setResource(DtoResourceStatus.builder()
                        .setName(TEST_HOST_NAME)
                        .setType(HOST_RESOURCE_TYPE)
                        .setStatus(DtoMonitorStatus.HOST_UP)
                        .setLastCheckTime(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                        .build())
                .build());

        resources.add(DtoResourceWithMetrics.builder()
                .setMetrics(timeSeries)
                .setResource(DtoResourceStatus.builder()
                        .setName(TEST_SERVICE_NAME)
                        .setType(SERVICE_RESOURCE_TYPE)
                        .setStatus(DtoMonitorStatus.SERVICE_OK)
                        .setLastCheckTime(new SimpleDateFormat("dd/MM/yyyy").parse("22/10/2019"))
                        .setOwner(TEST_HOST_NAME)
                        .build())
                .build());

        transit.SendResourcesWithMetrics(resources);

//        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
//        String name = reader.readLine();
    }


    /**
     * Test synchronize inventory
     * <p>
     * Usage:
     * 1. Start GroundWork Foundation server
     * 2. Generate '.so' file by running `go build -o libtransit/libtransit.so -buildmode=c-shared libtransit/libtransit.go` command in TNG
     * 4. Set path to '.so' file in TransitServiceImpl constructor.
     * 3. Run test
     */
    @Test
    public void shouldSynchronizeInventory() throws IOException {
        TransitServices transit = new TransitServicesImpl();

        transit.StartNATS();

        transit.StartNATS();

        transit.StartNATS();

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
        DtoInventoryResource service = DtoInventoryResource.builder()
                .setName(TEST_SERVICE_NAME)
                .setType(SERVICE_RESOURCE_TYPE)
                .setOwner(TEST_HOST_NAME)
                .build();

        DtoGroup group = new DtoGroup();
        List<DtoMonitoredResource> resources = new ArrayList<>();
        group.setGroupName("GW8");
        group.setResources(resources);

        DtoInventory dtoInventory = new DtoInventory();
        dtoInventory.setContext(context);
        dtoInventory.add(host);
        dtoInventory.add(service);
        dtoInventory.add(group);

        transit.SynchronizeInventory(dtoInventory);

//        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
//        String name = reader.readLine();
    }
}
