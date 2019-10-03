package org.groundwork.tng.transit;

import com.sun.jna.ptr.ByReference;

public class StringByReference extends ByReference {
    public StringByReference() {
        this(0);
    }

    public StringByReference(int size) {
        super(Math.max(size, 4));
    }

    public StringByReference(String str) {
        super(str.length() < 4 ? 4 : str.length() + 1);
        setValue(str);
    }

    private void setValue(String str) {
        getPointer().setString(0, str);
    }

    public String getValue() {
        return getPointer().getString(0);
    }
}