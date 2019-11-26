package serverconnector

import (
    "fmt"
    "github.com/gwos/tng/milliseconds"
    "github.com/gwos/tng/transit"
    "github.com/shirou/gopsutil/cpu"
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/host"
    "github.com/shirou/gopsutil/mem"
    "github.com/shirou/gopsutil/net"
    "runtime"
    "strconv"
    "time"
)

func CollectMetrics() transit.MonitoredResource {
    // TODO: BUILD SYSTEMSTATS and return
    runtimeOS := runtime.GOOS
    // memory
    vmStat, err := mem.VirtualMemory()
    dealwithErr(err)

    // disk - start from "/" mount point for Linux
    // might have to change for Windows!!
    // don't have a Window to test this out, if detect OS == windows
    // then use "\" instead of "/"

    diskStat, err := disk.Usage("/")
    dealwithErr(err)

    // cpu - get CPU number of cores and speed
    cpuStat, err := cpu.Info()
    dealwithErr(err)
    percentage, err := cpu.Percent(0, true)
    dealwithErr(err)

    // host or machine kernel, uptime, platform Info
    hostStat, err := host.Info()
    dealwithErr(err)

    // get interfaces MAC/hardware address
    interfStat, err := net.Interfaces()
    dealwithErr(err)

    // TODO: VLAD: remove below between ====== when done, its just examples
    // ====================================================================
    html := "<html>OS : " + runtimeOS + "<br>"
    html = html + "Total memory: " + strconv.FormatUint(vmStat.Total, 10) + " bytes <br>"
    html = html + "Free memory: " + strconv.FormatUint(vmStat.Free, 10) + " bytes<br>"
    html = html + "Percentage used memory: " + strconv.FormatFloat(vmStat.UsedPercent, 'f', 2, 64) + "%<br>"

    // get disk serial number.... strange... not available from disk package at compile time
    // undefined: disk.GetDiskSerialNumber
    //serial := disk.GetDiskSerialNumber("/dev/sda")

    //html = html + "Disk serial number: " + serial + "<br>"

    html = html + "Total disk space: " + strconv.FormatUint(diskStat.Total, 10) + " bytes <br>"
    html = html + "Used disk space: " + strconv.FormatUint(diskStat.Used, 10) + " bytes<br>"
    html = html + "Free disk space: " + strconv.FormatUint(diskStat.Free, 10) + " bytes<br>"
    html = html + "Percentage disk space usage: " + strconv.FormatFloat(diskStat.UsedPercent, 'f', 2, 64) + "%<br>"

    // since my machine has one CPU, I'll use the 0 index
    // if your machine has more than 1 CPU, use the correct index
    // to get the proper data
    html = html + "CPU index number: " + strconv.FormatInt(int64(cpuStat[0].CPU), 10) + "<br>"
    html = html + "VendorID: " + cpuStat[0].VendorID + "<br>"
    html = html + "Family: " + cpuStat[0].Family + "<br>"
    html = html + "Number of cores: " + strconv.FormatInt(int64(cpuStat[0].Cores), 10) + "<br>"
    html = html + "Model Name: " + cpuStat[0].ModelName + "<br>"
    html = html + "Speed: " + strconv.FormatFloat(cpuStat[0].Mhz, 'f', 2, 64) + " MHz <br>"

    for idx, cpupercent := range percentage {
        html = html + "Current CPU utilization: [" + strconv.Itoa(idx) + "] " + strconv.FormatFloat(cpupercent, 'f', 2, 64) + "%<br>"
    }

    html = html + "Hostname: " + hostStat.Hostname + "<br>"
    html = html + "Uptime: " + strconv.FormatUint(hostStat.Uptime, 10) + "<br>"
    html = html + "Number of processes running: " + strconv.FormatUint(hostStat.Procs, 10) + "<br>"

    // another way to get the operating system name
    // both darwin for Mac OSX, For Linux, can be ubuntu as platform
    // and linux for OS

    html = html + "OS: " + hostStat.OS + "<br>"
    html = html + "Platform: " + hostStat.Platform + "<br>"

    // the unique hardware id for this machine
    html = html + "Host ID(uuid): " + hostStat.HostID + "<br>"

    for _, interf := range interfStat {
        html = html + "------------------------------------------------------<br>"
        html = html + "Interface Name: " + interf.Name + "<br>"

        if interf.HardwareAddr != "" {
            html = html + "Hardware(MAC) Address: " + interf.HardwareAddr + "<br>"
        }

        for _, flag := range interf.Flags {
            html = html + "Interface behavior or flags: " + flag + "<br>"
        }

        for _, addr := range interf.Addrs {
            html = html + "IPv6 or IPv4 addresses: " + addr.String() + "<br>"

        }

    }
    // ======================================================================

    now := milliseconds.MillisecondTimestamp{Time: time.Now()}

    diskFree := transit.TimeSeries{
        MetricName: "diskFree",
        MetricSamples: []*transit.MetricSample{
            {
                SampleType: transit.Value,
                Interval:   &transit.TimeInterval{EndTime: now, StartTime: now},
                Value:      &transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(diskStat.Free)},
            },
        },
    }
    // TODO: we may want to convert from int64 to uint64
    // TODO: GET TOTAL DISK USAGE -- see above
    // TODO: GET TOTAL USED -- see above
    // TODO: GET PERCENT FREE -- see above

    // TODO: GET MEMORY USAGE -- see above -- total, used, free, percent
    // TODO: GET CPU STATS AND AVERAGE LOAD ACROSS ALL CPUS -- see above
    // TODO: advanced - one CPU service, multiple metrics per CPU1, CPU2 ... (probably not very useful)
    // TODO: hostStat.Procs -- number of processes running

    diskFreeService := transit.MonitoredService{
        Name:             "diskFree",
        Status:           transit.ServiceOk,
        LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
        Metrics: []transit.TimeSeries{diskFree},
    }

    return transit.MonitoredResource{
        Name:             hostStat.Hostname,
        Type:             transit.Host,
        Status:           transit.HostUp,
        LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
        Services: []transit.MonitoredService{diskFreeService},
    }
}

func dealwithErr(err error) {
    if err != nil {
        fmt.Println(err)
        //os.Exit(-1)
    }
}
