package org.groundwork.tcg.transit;

import com.sun.jna.Library;

public interface TcgTransitLibrary extends Library {
    public boolean GoSetenv(String key, String value, StringByReference errorMsg, Integer errorMsgSize);

    public boolean SendResourcesWithMetrics(String resourceWithMetricsJson, StringByReference errorMsg, Integer errorMsgSize);

    public boolean SynchronizeInventory(String inventoryJson, StringByReference errorMsg, Integer errorMsgSize);

    public boolean StartNats(StringByReference errorMsg, Integer errorMsgSize);

    public void StopNats();

    public boolean StartTransport(StringByReference errorMsg, Integer errorMsgSize);

    public boolean StopTransport(StringByReference errorMsg, Integer errorMsgSize);

    public boolean IsControllerRunning() throws TransitException;

    public boolean IsNATSRunning() throws TransitException;

    public boolean IsTransportRunning() throws TransitException;

    public void RegisterListMetricsHandler(ListMetricsCallback fn);

    public void RemoveListMetricsHandler();
}
