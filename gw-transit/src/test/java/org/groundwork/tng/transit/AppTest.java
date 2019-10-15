//package org.groundwork.tng.transit;
//
//import static org.junit.Assert.assertEquals;
//import static org.junit.Assert.assertTrue;
//
//import org.groundwork.rs.dto.DtoOperationResults;
//import org.groundwork.rs.transit.*;
//import org.junit.Test;
//
//import java.io.BufferedReader;
//import java.io.IOException;
//import java.io.InputStreamReader;
//import java.util.ArrayList;
//import java.util.Date;
//import java.util.List;
//
///**
// * Unit test for simple App.
// */
//public class AppTest {
//    /**
//     * Test sending resource and metrics
//     * <p>
//     * Usage:
//     * 1. Start GroundWork Foundation server
//     * 2. Generate '.so' file by running `go build -o libtransit.so -buildmode=c-shared libtransit.go` command from
//     * libtransit package on TNG
//     * 4. Set path to '.so' file in TransitServiceImpl constructor.
//     * 3. Run test
//     */
//    @Test
//    public void shouldSendResourceAndMetrics() throws IOException {
//        TransitServices transit = new TransitServicesImpl();
//
//        List<DtoTimeSeries> timeSeries = new ArrayList<>();
//        timeSeries.add(DtoTimeSeries.builder()
//                .setMetricName("mc-test-service-0")
//                .setSampleType(DtoMetricSampleType.Warning)
//                .setInterval(DtoTimeInterval.builder()
//                        .setStartTime(new Date())
//                        .setEndTime(new Date())
//                        .build())
//                .setValue(DtoTypedValue.builder()
//                        .setValueType(DtoValueType.IntegerType)
//                        .setIntegerValue(1)
//                        .build())
//                .build());
//
//        DtoResourceWithMetricsList resources = DtoResourceWithMetricsList.builder()
//                .setContext(DtoTracerContext.builder()
//                        .setAgentId("3939333393342")
//                        .setAppType("VEMA")
//                        .setTimeStamp(new Date())
//                        .setTraceToken("token-99e93")
//                        .build())
//                .build();
//
//
//        resources.add(DtoResourceWithMetrics.builder()
//                .setMetrics(timeSeries)
//                .setResource(DtoMonitoredResource.builder()
//                        .setName("mc-test-host")
//                        .setType("HOST")
//                        .setStatus(DtoMonitorStatus.HOST_UP)
//                        .setOwner("mc-test-host")
//                        .build())
//                .build());
//
//        transit.SendResourcesWithMetrics(resources);
//
//        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
//        String name = reader.readLine();
//
//        transit.Disconnect();
//
////        assertEquals(1, (int) results.getCount());
////        assertEquals(0, (int) results.getSuccessful());
////        assertEquals(1, (int) results.getFailed());
//    }
//
//
//    /**
//     * Test synchronize inventory
//     * <p>
//     * Usage:
//     * 1. Start GroundWork Foundation server
//     * 2. Generate '.so' file by running `go build -o libtransit.so -buildmode=c-shared libtransit.go` command from
//     * libtransit package on TNG
//     * 4. Set path to '.so' file in TransitServiceImpl constructor.
//     * 3. Run test
//     */
//    @Test
//    public void shouldSynchronizeInventory() throws IOException {
//        TransitServices transit = new TransitServicesImpl();
//
//        DtoTracerContext context = DtoTracerContext.builder()
//                .setAgentId("3939333393342")
//                .setAppType("VEMA")
//                .setTimeStamp(new Date())
//                .setTraceToken("token-99e93")
//                .build();
//
//        DtoMonitoredResource resource = DtoMonitoredResource.builder()
//                .setName("mc-test-host")
//                .setType("HOST")
//                .setStatus(DtoMonitorStatus.HOST_UP)
//                .setOwner("mc-test-host")
//                .build();
//
//        DtoGroup group = new DtoGroup();
//        List<DtoMonitoredResource> resources = new ArrayList<>();
//        group.setGroupName("GW8");
//        group.setResources(resources);
//
//        DtoInventory dtoInventory = new DtoInventory();
//        dtoInventory.setContext(context);
//        dtoInventory.add(resource);
//        dtoInventory.add(group);
//
//        transit.SynchronizeInventory(dtoInventory);
//
//        BufferedReader reader = new BufferedReader(new InputStreamReader(System.in));
//        String name = reader.readLine();
//
//        transit.Disconnect();
//    }
//}
