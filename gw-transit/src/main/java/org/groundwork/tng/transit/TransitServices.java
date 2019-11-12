package org.groundwork.tng.transit;

import org.groundwork.rs.transit.DtoInventory;
import org.groundwork.rs.transit.DtoResourceWithMetricsList;


public interface TransitServices {

    void SendResourcesWithMetrics(DtoResourceWithMetricsList resources) throws TransitException;

    void SynchronizeInventory(DtoInventory inventory) throws TransitException;

    void StartNATS() throws TransitException;

    void StopNATS() throws TransitException;

    void StartTransport() throws TransitException;

    void StopTransport() throws TransitException;

    boolean IsControllerRunning() throws TransitException;

    boolean IsNATSRunning() throws TransitException;

    boolean IsTransportRunning() throws TransitException;

    void RegisterListMetricsHandler(ListMetricsCallback callback) throws TransitException;

    void RemoveListMetricsHandler() throws TransitException;
}
