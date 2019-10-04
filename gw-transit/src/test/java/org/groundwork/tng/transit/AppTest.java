package org.groundwork.tng.transit;

import static org.junit.Assert.assertTrue;

import org.groundwork.rs.dto.DtoOperationResults;
import org.groundwork.rs.transit.DtoMonitoredResource;
import org.groundwork.rs.transit.DtoResourceWithMetrics;
import org.groundwork.rs.transit.DtoResourceWithMetricsList;
import org.groundwork.rs.transit.DtoTimeSeries;
import org.groundwork.rs.transit.DtoTracerContext;
import org.junit.Test;

import java.util.ArrayList;
import java.util.Date;
import java.util.List;

/**
 * Unit test for simple App.
 */
public class AppTest 
{
    /**
     * Test sending resource and metrics
     */
    @Test
    public void shouldSendResourceAndMetrics() {
        TransitServices transit = new TransitServicesImpl();
        DtoResourceWithMetricsList resources = new DtoResourceWithMetricsList();
        // TODO: constructor for tracer context or Builder Pattern
        DtoTracerContext tracer = new DtoTracerContext();
        tracer.setAgentId("3939333393342");
        tracer.setAppType("VEMA");
        tracer.setTimeStamp(new Date());
        tracer.setTraceToken("token-99e93");
        resources.setContext(tracer);
        // TODO: Builder pattern
        DtoResourceWithMetrics resourceWithMetrics = new DtoResourceWithMetrics();
        // TODO: Builder pattern
        DtoMonitoredResource resource = new DtoMonitoredResource();

        // TODO: builder pattern
        List<DtoTimeSeries> timeSeries = new ArrayList<>();
        DtoTimeSeries ts1 = new DtoTimeSeries();

        resourceWithMetrics.setResource(resource);
        resourceWithMetrics.setMetrics(timeSeries);
        resources.add(resourceWithMetrics);

        DtoOperationResults results = transit.SendResourcesWithMetrics(resources);
        assertTrue( results.getCount().equals(1) );
        assertTrue( results.getSuccessful().equals(1) );
        assertTrue( results.getFailed().equals(0) );
    }
}
