package com.swingtrade.api.config;

import com.swingtrade.api.dto.ErrorResponse;
import org.junit.jupiter.api.Test;
import org.springframework.core.MethodParameter;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.web.context.request.WebRequest;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;

import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

/**
 * Unit tests for GlobalExceptionHandler setup.
 * Note: Actual exception handler tests will be integration tests using MockMvc.
 * This file tests that the handler can be instantiated and error response methods work.
 */
class GlobalExceptionHandlerTest {

    @Test
    void testGlobalExceptionHandlerInstantiates() {
        // Arrange & Act
        GlobalExceptionHandler handler = new GlobalExceptionHandler();

        // Assert
        assertNotNull(handler);
    }

    @Test
    void testErrorResponseStaticFactoryMethods() {
        // Test badRequest
        ErrorResponse badRequest = ErrorResponse.badRequest("Validation failed");
        assertEquals(400, badRequest.getStatus());
        assertEquals("BAD_REQUEST", badRequest.getCode());
        assertEquals("Validation failed", badRequest.getMessage());
        assertNotNull(badRequest.getTimestamp());

        // Test notFound
        ErrorResponse notFound = ErrorResponse.notFound("Stock not found");
        assertEquals(404, notFound.getStatus());
        assertEquals("NOT_FOUND", notFound.getCode());
        assertEquals("Stock not found", notFound.getMessage());
        assertNotNull(notFound.getTimestamp());

        // Test conflict
        ErrorResponse conflict = ErrorResponse.conflict("Position already exists");
        assertEquals(409, conflict.getStatus());
        assertEquals("CONFLICT", conflict.getCode());
        assertEquals("Position already exists", conflict.getMessage());
        assertNotNull(conflict.getTimestamp());

        // Test internalError
        ErrorResponse internalError = ErrorResponse.internalError("Database error");
        assertEquals(500, internalError.getStatus());
        assertEquals("INTERNAL_ERROR", internalError.getCode());
        assertEquals("Database error", internalError.getMessage());
        assertNotNull(internalError.getTimestamp());
    }

    @Test
    void testErrorResponseStaticFactoryMethodsWithPath() {
        // Test badRequest with path
        ErrorResponse badRequest = ErrorResponse.badRequest("Validation failed", "/api/stocks");
        assertEquals(400, badRequest.getStatus());
        assertEquals("BAD_REQUEST", badRequest.getCode());
        assertEquals("/api/stocks", badRequest.getPath());

        // Test notFound with path
        ErrorResponse notFound = ErrorResponse.notFound("Stock not found", "/api/stocks/123");
        assertEquals(404, notFound.getStatus());
        assertEquals("NOT_FOUND", notFound.getCode());
        assertEquals("/api/stocks/123", notFound.getPath());

        // Test conflict with path
        ErrorResponse conflict = ErrorResponse.conflict("Position exists", "/api/positions");
        assertEquals(409, conflict.getStatus());
        assertEquals("CONFLICT", conflict.getCode());
        assertEquals("/api/positions", conflict.getPath());

        // Test internalError with path
        ErrorResponse internalError = ErrorResponse.internalError("Server error", "/api/internal");
        assertEquals(500, internalError.getStatus());
        assertEquals("INTERNAL_ERROR", internalError.getCode());
        assertEquals("/api/internal", internalError.getPath());
    }

    @Test
    void testHandleTypeMismatchReturnsBadRequest() throws NoSuchMethodException {
        GlobalExceptionHandler handler = new GlobalExceptionHandler();
        WebRequest request = mock(WebRequest.class);
        when(request.getDescription(false)).thenReturn("uri=/api/job/runs/not-a-uuid/progress");

        MethodParameter param = new MethodParameter(
                GlobalExceptionHandlerTest.class.getDeclaredMethod("testHandleTypeMismatchReturnsBadRequest"), -1);
        MethodArgumentTypeMismatchException ex = new MethodArgumentTypeMismatchException(
                "not-a-uuid", UUID.class, "runId", param, new IllegalArgumentException("Invalid UUID"));

        ResponseEntity<ErrorResponse> response = handler.handleTypeMismatch(ex, request);

        assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
        assertNotNull(response.getBody());
        assertEquals(400, response.getBody().getStatus());
        assertEquals("VALIDATION_ERROR", response.getBody().getCode());
        assertEquals("/api/job/runs/not-a-uuid/progress", response.getBody().getPath());
    }

    @Test
    void testHandleIllegalArgumentReturnsBadRequest() {
        GlobalExceptionHandler handler = new GlobalExceptionHandler();
        WebRequest request = mock(WebRequest.class);
        when(request.getDescription(false)).thenReturn("uri=/api/positions");

        IllegalArgumentException ex = new IllegalArgumentException("Price is required and must be greater than zero");

        ResponseEntity<ErrorResponse> response = handler.handleIllegalArgument(ex, request);

        assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
        assertNotNull(response.getBody());
        assertEquals(400, response.getBody().getStatus());
        assertEquals("VALIDATION_ERROR", response.getBody().getCode());
        assertEquals("Price is required and must be greater than zero", response.getBody().getMessage());
    }

    @Test
    void testHandleUnreadableMessageReturnsBadRequest() {
        GlobalExceptionHandler handler = new GlobalExceptionHandler();
        WebRequest request = mock(WebRequest.class);
        when(request.getDescription(false)).thenReturn("uri=/api/positions");

        HttpMessageNotReadableException ex = mock(HttpMessageNotReadableException.class);
        when(ex.getMessage()).thenReturn("Cannot deserialize enum value");

        ResponseEntity<ErrorResponse> response = handler.handleUnreadableMessage(ex, request);

        assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
        assertNotNull(response.getBody());
        assertEquals(400, response.getBody().getStatus());
        assertEquals("VALIDATION_ERROR", response.getBody().getCode());
        assertEquals("Request body is invalid or unreadable", response.getBody().getMessage());
        assertEquals("/api/positions", response.getBody().getPath());
    }
}
