package org.groundwork.tng.transit;


import com.sun.jna.Library;

public interface TngTransitLibrary extends Library {
    public String ListMetrics(StringByReference errorMsg);
    public boolean SendResourcesWithMetrics(String resourceWithMetricsJson, StringByReference errorMsg);
    public boolean SynchronizeInventory(String inventoryJson, StringByReference errorMsg);
    public void ListInventory(StringByReference errorMsg);
    public boolean StartNATS(StringByReference errorMsg);
    public void StopNATS();
    public boolean StartTransport(StringByReference errorMsg);
    public boolean StopTransport(StringByReference errorMsg);
}
