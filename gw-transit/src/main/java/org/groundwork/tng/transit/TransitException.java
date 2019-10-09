package org.groundwork.tng.transit;

public class TransitException extends RuntimeException {
    public TransitException() {
    }

    public TransitException(String message) {
        super(message);
    }

    public TransitException(String message, Throwable cause) {
        super(message, cause);
    }

    public TransitException(Throwable cause) {
        super(cause);
    }

}
