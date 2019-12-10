package org.groundwork.tng.transit;

import org.groundwork.rs.transit.DtoInventory;
import org.groundwork.rs.transit.DtoResourcesWithServices;


public interface TransitServices {
    boolean GoSetenv(String key, String value, StringByReference errorMsg, Integer errorMsgSize);

    void SendResourcesWithMetrics(DtoResourcesWithServices resources) throws TransitException;

    void SynchronizeInventory(DtoInventory inventory) throws TransitException;

    void StartNats() throws TransitException;

    void StopNats() throws TransitException;

    void StartTransport() throws TransitException;

    void StopTransport() throws TransitException;

    boolean IsControllerRunning() throws TransitException;

    boolean IsNATSRunning() throws TransitException;

    boolean IsTransportRunning() throws TransitException;

    void RegisterListMetricsHandler(ListMetricsCallback callback) throws TransitException;

    void RemoveListMetricsHandler() throws TransitException;
}
