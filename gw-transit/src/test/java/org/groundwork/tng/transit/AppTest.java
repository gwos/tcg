package org.groundwork.tng.transit;

import org.groundwork.rs.transit.*;
import org.junit.Test;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

/**
 * Unit test for simple App.
 */
public class AppTest {
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
    public void shouldSendResourceAndMetrics() throws IOException {
        TransitServices transit = new TransitServicesImpl();

        DtoTracerContext context = DtoTracerContext.builder()
                .setAgentId("3939333393342")
                .setAppType("VEMA")
                .setTimeStamp(new Date())
                .setTraceToken("token-99e93")
                .build();

        List<DtoTimeSeries> timeSeries = new ArrayList<>();
        timeSeries.add(DtoTimeSeries.builder()
                .setMetricName("mc-test-service-0")
                .setSampleType(DtoMetricSampleType.Warning)
                .setInterval(DtoTimeInterval.builder()
                        .setStartTime(new Date())
                        .setEndTime(new Date())
                        .build())
                .setValue(DtoTypedValue.builder()
                        .setValueType(DtoValueType.IntegerType)
                        .setIntegerValue(1)
                        .build())
                .build());

        DtoResourceWithMetricsList resources = DtoResourceWithMetricsList.builder().setContext(context).build();

        resources.add(DtoResourceWithMetrics.builder()
                .setResource(DtoResourceStatus.builder()
                        .setName("mc-test-host")
                        .setType("HOST")
                        .setStatus(DtoMonitorStatus.HOST_UP)
                        .build())
                .build());

        resources.add(DtoResourceWithMetrics.builder()
                .setMetrics(timeSeries)
                .setResource(DtoResourceStatus.builder()
                        .setName("mc-test-service-0")
                        .setType("SERVICE")
                        .setStatus(DtoMonitorStatus.SERVICE_OK)
                        .setOwner("mc-test-host")
                        .build())
                .build());

        transit.SendResourcesWithMetrics(resources);

        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
        String name = reader.readLine();
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
                .setName("mc-test-host")
                .setType("HOST") // TODO: use constant
                .build();
        DtoInventoryResource service = DtoInventoryResource.builder()
                .setName("mc-test-service-0")
                .setType("SERVICE") // TODO: use constant
                .setOwner("mc-test-host")
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

        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
        String name = reader.readLine();
    }
}
