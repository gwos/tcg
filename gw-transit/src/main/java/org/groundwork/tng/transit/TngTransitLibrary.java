package org.groundwork.tng.transit;


import com.sun.jna.Library;

public interface TngTransitLibrary extends Library {
    public String SendMetrics(String metric, StringByReference errorMsg);
    public String ListMetrics(StringByReference errorMsg);
    public boolean SendResourcesWithMetrics(String resourceWithMetricsJson, StringByReference errorMsg);
    public boolean SynchronizeInventory(String inventoryJson, StringByReference errorMsg);
    public void ListInventory(StringByReference errorMsg);
    public boolean Connect(String credentialsJson, StringByReference errorMsg);
    public boolean Disconnect(StringByReference errorMsg);
}
