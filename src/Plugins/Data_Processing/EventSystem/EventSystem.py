"""
EventSystem module.

A generic event system for managing subscriptions and notifications between components.
"""

from typing import Dict, List, Any, Callable
import logging
from Plugins.BasePlugin import BasePlugin


class EventSystem(BasePlugin):
    """
    Generic event system plugin for managing subscriptions and event dispatching.
    Provides functionality for:
    - Event subscription/unsubscription
    - Event dispatching
    - Filtered subscriptions
    - Event history tracking
    """

    plugin_type = "Data_Processing"

    def __init__(self):
        """Initialize the EventSystem plugin"""
        self.subscribers = {}  # Event type -> list of subscriber IDs
        self.callbacks = {}    # Subscriber ID -> callback info
        self.history = {}      # Event type -> list of recent events
        self.history_limit = 100
        self.logger = logging.getLogger(__name__)

    def subscribe(self, subscriber_id: str, event_type: str, filter_fn: Callable = None) -> None:
        """Subscribe to an event type with optional filtering
        
        Args:
            subscriber_id: Unique identifier for the subscriber
            event_type: Type of event to subscribe to
            filter_fn: Optional function to filter events
        """
        if event_type not in self.subscribers:
            self.subscribers[event_type] = []
        
        if subscriber_id not in self.subscribers[event_type]:
            self.subscribers[event_type].append(subscriber_id)
            
        self.callbacks[subscriber_id] = {
            "event_type": event_type,
            "filter": filter_fn
        }
        
        self.logger.info(f"Added subscription: {subscriber_id} -> {event_type}")

    def unsubscribe(self, subscriber_id: str, event_type: str = None) -> None:
        """Unsubscribe from events
        
        Args:
            subscriber_id: ID of subscriber to remove
            event_type: Optional event type, if None removes all subscriptions
        """
        if event_type:
            if event_type in self.subscribers:
                if subscriber_id in self.subscribers[event_type]:
                    self.subscribers[event_type].remove(subscriber_id)
                    if subscriber_id in self.callbacks:
                        del self.callbacks[subscriber_id]
        else:
            # Remove all subscriptions for this subscriber
            for evt_type in self.subscribers:
                if subscriber_id in self.subscribers[evt_type]:
                    self.subscribers[evt_type].remove(subscriber_id)
            if subscriber_id in self.callbacks:
                del self.callbacks[subscriber_id]
                
        self.logger.info(f"Removed subscription: {subscriber_id}")

    def dispatch_event(self, event_type: str, event_data: Dict[str, Any]) -> List[str]:
        """Dispatch an event to all relevant subscribers
        
        Args:
            event_type: Type of event being dispatched
            event_data: Data payload for the event
            
        Returns:
            List of subscriber IDs that received the event
        """
        if event_type not in self.subscribers:
            return []

        # Add to history
        if event_type not in self.history:
            self.history[event_type] = []
        self.history[event_type].append(event_data)
        if len(self.history[event_type]) > self.history_limit:
            self.history[event_type].pop(0)

        # Notify subscribers
        notified = []
        for subscriber_id in self.subscribers[event_type]:
            callback_info = self.callbacks.get(subscriber_id)
            if callback_info:
                filter_fn = callback_info.get("filter")
                if filter_fn is None or filter_fn(event_data):
                    notified.append(subscriber_id)

        self.logger.info(f"Dispatched {event_type} event to {len(notified)} subscribers")
        return notified

    def get_event_history(self, event_type: str = None) -> Dict[str, List[Dict[str, Any]]]:
        """Get history of events
        
        Args:
            event_type: Optional type to filter history
            
        Returns:
            Dictionary of event types and their history
        """
        if event_type:
            return {event_type: self.history.get(event_type, [])}
        return self.history

    def clear_history(self, event_type: str = None) -> None:
        """Clear event history
        
        Args:
            event_type: Optional type to clear, if None clears all
        """
        if event_type:
            if event_type in self.history:
                self.history[event_type] = []
        else:
            self.history = {}

    def execute_pipeline_step(self, step_config: dict, context: dict) -> dict:
        """Execute a pipeline step for this plugin
        
        Args:
            step_config: Configuration dictionary containing:
                - operation: Type of operation to perform
                    - 'subscribe': Add new subscription
                    - 'unsubscribe': Remove subscription
                    - 'dispatch': Dispatch an event
                    - 'history': Get event history
                    - 'clear': Clear event history
                - subscriber_id: ID for subscribe/unsubscribe
                - event_type: Type of event
                - event_data: Data for dispatch operation
                - filter: Optional filter function for subscriptions
            context: Pipeline context
            
        Returns:
            Dictionary containing operation results
        """
        config = step_config.get("config", {})
        operation = config.get("operation")
        
        try:
            if operation == "subscribe":
                subscriber_id = config["subscriber_id"]
                event_type = config["event_type"]
                filter_fn = config.get("filter")
                self.subscribe(subscriber_id, event_type, filter_fn)
                return {step_config["output"]: {"subscribed": True}}
                
            elif operation == "unsubscribe":
                subscriber_id = config["subscriber_id"]
                event_type = config.get("event_type")
                self.unsubscribe(subscriber_id, event_type)
                return {step_config["output"]: {"unsubscribed": True}}
                
            elif operation == "dispatch":
                event_type = config["event_type"]
                event_data = config["event_data"]
                notified = self.dispatch_event(event_type, event_data)
                return {step_config["output"]: {
                    "dispatched": True,
                    "notified": notified
                }}
                
            elif operation == "history":
                event_type = config.get("event_type")
                history = self.get_event_history(event_type)
                return {step_config["output"]: history}
                
            elif operation == "clear":
                event_type = config.get("event_type")
                self.clear_history(event_type)
                return {step_config["output"]: {"cleared": True}}
                
            else:
                raise ValueError(f"Unknown operation: {operation}")
                
        except Exception as e:
            raise ValueError(f"Error in EventSystem plugin: {str(e)}")