package org.groundwork.tng.transit;

import com.sun.jna.Library;

public interface TngTransitLibrary extends Library {
    public boolean SendResourcesWithMetrics(String resourceWithMetricsJson, StringByReference errorMsg);
    public boolean SynchronizeInventory(String inventoryJson, StringByReference errorMsg);
    public boolean StartNATS(StringByReference errorMsg);
    public void StopNATS();
    public boolean StartTransport(StringByReference errorMsg);
    public boolean StopTransport(StringByReference errorMsg);
    public boolean IsControllerRunning() throws TransitException;
    public boolean IsNATSRunning() throws TransitException;
    public boolean IsTransportRunning() throws TransitException;
    public void RegisterListMetricsHandler(ListMetricsCallback fn);
    public void RemoveListMetricsHandler();
}
