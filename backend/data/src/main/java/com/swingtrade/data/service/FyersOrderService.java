package com.swingtrade.data.service;

import com.tts.in.model.OrderModel;
import com.tts.in.model.PlaceOrderModel;
import com.tts.in.model.FyersClass;
import org.json.JSONObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.function.Supplier;

/**
 * Fyers v3 order management service.
 */
public class FyersOrderService {

    private static final Logger logger = LoggerFactory.getLogger(FyersOrderService.class);
    private final FyersAuthService authService;
    private final Supplier<FyersClass> sdkProvider;
    private volatile boolean missingTokenLogged = false;

    public FyersOrderService(FyersAuthService authService) {
        this(authService, FyersClass::getInstance);
    }

    FyersOrderService(FyersAuthService authService, Supplier<FyersClass> sdkProvider) {
        this.authService = authService;
        this.sdkProvider = sdkProvider;
    }

    private FyersClass getSdk() {
        FyersClass sdk = sdkProvider.get();
        sdk.clientId = authService.getClientId();
        String token = authService.getAccessToken();
        if (token != null) sdk.accessToken = token;
        return sdk;
    }

    public String placeOrder(String symbol, int qty, int side, int orderType,
                              double limitPrice, String productType,
                              Double stopLoss, Double takeProfit, String orderTag) {
        String token = authService.getAccessToken();
        if (token == null) {
            if (!missingTokenLogged) {
                logger.warn("Fyers access token is missing. Order placement skipped.");
                missingTokenLogged = true;
            }
            return null;
        }

        PlaceOrderModel model = new PlaceOrderModel();
        model.Symbol = symbol;
        model.Qty = qty;
        model.Side = side;
        model.OrderType = orderType;
        model.LimitPrice = limitPrice;
        model.ProductType = productType;
        model.OrderValidity = "DAY";
        if (stopLoss != null) model.StopLoss = stopLoss;
        if (takeProfit != null) model.TakeProfit = takeProfit;
        if (orderTag != null) model.OrderTag = orderTag;

        try {
            com.tts.in.utilities.Tuple<JSONObject, JSONObject> result = getSdk().PlaceOrder(model);
            if (result == null) return null;
            JSONObject metadata = result.Item1();
            if (!"success".equals(metadata.optString("s"))) {
                logger.error("Place order failed: {}", metadata.optString("message", "unknown"));
                return null;
            }
            JSONObject body = result.Item2();
            return body.has("id") ? body.getString("id") : null;
        } catch (Exception e) {
            logger.error("Place order error: {}", e.getMessage());
            return null;
        }
    }

    public boolean cancelOrder(String orderId) {
        try {
            com.tts.in.utilities.Tuple<JSONObject, JSONObject> result = getSdk().CancelOrder(orderId);
            if (result == null) return false;
            return "success".equals(result.Item1().optString("s"));
        } catch (Exception e) {
            logger.error("Cancel order error: {}", e.getMessage());
            return false;
        }
    }

    public List<OrderModel> getAllOrders() {
        try {
            com.tts.in.utilities.Tuple<JSONObject, JSONObject> result = getSdk().GetAllOrders();
            if (result == null) return Collections.emptyList();
            if (!"success".equals(result.Item1().optString("s"))) return Collections.emptyList();
            return parseOrderList(result.Item2());
        } catch (Exception e) {
            logger.error("Get orders error: {}", e.getMessage());
            return Collections.emptyList();
        }
    }

    public List<OrderModel> getOrderHistory(String symbols) {
        try {
            com.tts.in.utilities.Tuple<JSONObject, JSONObject> result = getSdk().GetOrderHistory(symbols);
            if (result == null) return Collections.emptyList();
            if (!"success".equals(result.Item1().optString("s"))) return Collections.emptyList();
            return parseOrderList(result.Item2());
        } catch (Exception e) {
            logger.error("Get order history error: {}", e.getMessage());
            return Collections.emptyList();
        }
    }

    private List<OrderModel> parseOrderList(JSONObject data) {
        List<OrderModel> orders = new ArrayList<>();
        if (data == null) return orders;
        JSONObject nodes = data.has("orderBook") ? data.optJSONObject("orderBook") : data;
        if (nodes == null) return orders;
        for (int i = 0; i < nodes.length(); i++) {
            String key = Integer.toString(i);
            if (!nodes.has(key)) break;
            JSONObject node = nodes.optJSONObject(key);
            if (node == null) continue;
            OrderModel model = new OrderModel();
            model.OrderId = node.optString("id", null);
            model.Symbol = node.optString("symbol", null);
            model.Qty = node.optInt("qty", 0);
            model.RemainingQuantity = node.optInt("remainingQuantity", 0);
            model.FilledQty = node.optInt("filledQty", 0);
            model.LimitPrice = node.optDouble("limitPrice", 0);
            model.StopPrice = node.optDouble("stopPrice", 0);
            model.TradedPrice = node.optDouble("tradedPrice", 0);
            model.OrderType = node.optInt("type", 0);
            model.ProductType = node.optString("productType", null);
            model.Side = node.optInt("side", 0);
            model.OrderStatus = node.optInt("status", 0);
            model.OrderDateTime = node.optString("orderDateTime", null);
            model.OrderValidity = node.optString("orderValidity", null);
            model.OrderTag = node.optString("orderTag", null);
            model.Message = node.optString("message", null);
            orders.add(model);
        }
        return orders;
    }
}
