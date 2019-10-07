package org.groundwork.tng.transit;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import org.groundwork.rs.dto.DtoOperationResults;
import org.groundwork.rs.transit.*;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Date;
import java.util.List;

/**
 * Unit test for simple App.
 */
public class AppTest {
    /**
     * Test sending resource and metrics
     *
     * Usage:
     * 1. Start GroundWork Foundation server
     * 2. Generate GWOS-API-TOKEN and paste it to headers in SendResourceWithMetric function in transit package from TNG
     * 3. Generate '.so' file by running `go build -o libtransit.so -buildmode=c-shared libtransit.go` command from
     * libtransit package on TNG
     * 4. Set path to '.so' file in TransitServiceImpl constructor.
     * 3. Run test
     */
    @Test
    public void shouldSendResourceAndMetrics() {
        TransitServices transit = new TransitServicesImpl();

        DtoCredentials credentials = new DtoCredentials();
        credentials.setUser("RESTAPIACCESS");
        credentials.setPassword("***REMOVED***");

        transit.Connect(credentials);

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


        DtoResourceWithMetricsList resources = DtoResourceWithMetricsList.builder()
                .setContext(DtoTracerContext.builder()
                        .setAgentId("3939333393342")
                        .setAppType("VEMA")
                        .setTimeStamp(new Date())
                        .setTraceToken("token-99e93")
                        .build())
                .build();


        resources.add(DtoResourceWithMetrics.builder()
                .setMetrics(timeSeries)
                .setResource(DtoMonitoredResource.builder()
                        .setName("mc-test-host")
                        .setType("HOST")
                        .setStatus(DtoMonitorStatus.HOST_UP)
                        .setOwner("mc-test-host")
                        .build())
                .build());

        DtoOperationResults results = transit.SendResourcesWithMetrics(resources);

        transit.Disconnect();

        assertEquals(1, (int) results.getCount());
        assertEquals(0, (int) results.getSuccessful());
        assertEquals(1, (int) results.getFailed());
    }
}
