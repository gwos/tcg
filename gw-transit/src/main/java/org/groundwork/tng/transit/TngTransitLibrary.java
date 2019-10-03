package org.groundwork.tng.transit;


import com.sun.jna.Library;

public interface TngTransitLibrary extends Library {
    public String SendMetrics(String metric, StringByReference errorMsg);
    public String ListMetrics(StringByReference errorMsg);
    public String SendResourcesWithMetrics(String resourceWithMetricsJson, StringByReference errorMsg);
    public String SynchronizeInventory(String inventoryJson, String groupsJson, StringByReference errorMsg);
    public void ListInventory(StringByReference errorMsg);
    public String Connect(String credentialsJson, StringByReference errorMsg);
    public boolean Disconnect(String transitJson, StringByReference errorMsg);
}