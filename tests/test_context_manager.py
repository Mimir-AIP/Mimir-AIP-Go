import unittest
from unittest.mock import MagicMock
import timeit
from src.Plugins.Data_Processing.ContextManager.ContextManager import ContextManager
from src.Plugins.PluginManager import PluginManager

class TestContextManager(unittest.TestCase):
    def setUp(self):
        """Set up test environment before each test."""
        self.plugin_manager = MagicMock(spec=PluginManager)
        self.context_manager = ContextManager(self.plugin_manager)

    def tearDown(self):
        """Clean up after each test."""
        self.context_manager.clear_context()

    # Test get_context
    def test_get_full_context(self):
        """Test getting full context."""
        self.context_manager.set_context('key1', 'value1')
        self.context_manager.set_context('key2', 'value2')
        context = self.context_manager.get_context()
        self.assertEqual(context, {'key1': 'value1', 'key2': 'value2'})

    def test_get_specific_key(self):
        """Test getting specific context key."""
        self.context_manager.set_context('test_key', 'test_value')
        value = self.context_manager.get_context('test_key')
        self.assertEqual(value, 'test_value')

    def test_get_nonexistent_key(self):
        """Test getting non-existent key returns None."""
        value = self.context_manager.get_context('nonexistent')
        self.assertIsNone(value)

    # Test set_context
    def test_set_new_context(self):
        """Test setting new context key."""
        result = self.context_manager.set_context('new_key', 'new_value')
        self.assertTrue(result)
        self.assertEqual(self.context_manager.get_context('new_key'), 'new_value')

    def test_set_existing_context_overwrite_true(self):
        """Test overwriting existing context."""
        self.context_manager.set_context('key', 'old_value')
        result = self.context_manager.set_context('key', 'new_value', overwrite=True)
        self.assertTrue(result)
        self.assertEqual(self.context_manager.get_context('key'), 'new_value')

    def test_set_existing_context_overwrite_false(self):
        """Test not overwriting existing context."""
        self.context_manager.set_context('key', 'old_value')
        result = self.context_manager.set_context('key', 'new_value', overwrite=False)
        self.assertFalse(result)
        self.assertEqual(self.context_manager.get_context('key'), 'old_value')

    def test_set_invalid_key(self):
        """Test setting context with invalid key."""
        with self.assertRaises(ValueError):
            self.context_manager.set_context('', 'value')
        with self.assertRaises(ValueError):
            self.context_manager.set_context(None, 'value')

    # Test merge_context
    def test_merge_overwrite_strategy(self):
        """Test merge with overwrite strategy."""
        self.context_manager.set_context('key1', 'old1')
        self.context_manager.set_context('key2', 'old2')
        conflicts = self.context_manager.merge_context(
            {'key1': 'new1', 'key3': 'new3'}, 
            conflict_strategy='overwrite'
        )
        self.assertEqual(conflicts, {'key1': 'old1'})
        self.assertEqual(self.context_manager.get_context('key1'), 'new1')
        self.assertEqual(self.context_manager.get_context('key2'), 'old2')
        self.assertEqual(self.context_manager.get_context('key3'), 'new3')

    def test_merge_keep_strategy(self):
        """Test merge with keep strategy."""
        self.context_manager.set_context('key1', 'old1')
        conflicts = self.context_manager.merge_context(
            {'key1': 'new1'}, 
            conflict_strategy='keep'
        )
        self.assertEqual(conflicts, {'key1': 'new1'})
        self.assertEqual(self.context_manager.get_context('key1'), 'old1')

    def test_merge_merge_strategy_dicts(self):
        """Test merge strategy with dictionaries."""
        self.context_manager.set_context('key1', {'a': 1, 'b': 2})
        conflicts = self.context_manager.merge_context(
            {'key1': {'b': 3, 'c': 4}}, 
            conflict_strategy='merge'
        )
        self.assertEqual(conflicts, {'key1': {'a': 1, 'b': 2}})
        self.assertEqual(self.context_manager.get_context('key1'), 
                        {'a': 1, 'b': 3, 'c': 4})

    def test_merge_merge_strategy_non_dicts(self):
        """Test merge strategy with non-dictionaries."""
        self.context_manager.set_context('key1', 'value1')
        conflicts = self.context_manager.merge_context(
            {'key1': 'new_value'}, 
            conflict_strategy='merge'
        )
        self.assertEqual(conflicts, {'key1': 'value1'})
        self.assertEqual(self.context_manager.get_context('key1'), 'new_value')

    def test_merge_empty_context(self):
        """Test merging with empty context."""
        conflicts = self.context_manager.merge_context({'key1': 'value1'})
        self.assertEqual(conflicts, {})
        self.assertEqual(self.context_manager.get_context('key1'), 'value1')

    def test_merge_invalid_strategy(self):
        """Test merge with invalid strategy."""
        with self.assertRaises(ValueError):
            self.context_manager.merge_context({}, conflict_strategy='invalid')

    # Test snapshot and restore
    def test_snapshot_and_restore(self):
        """Test taking and restoring snapshots."""
        self.context_manager.set_context('key1', 'value1')
        snapshot_id = self.context_manager.snapshot_context()
        
        self.context_manager.set_context('key1', 'new_value')
        self.context_manager.set_context('key2', 'value2')
        
        result = self.context_manager.restore_context(snapshot_id)
        self.assertTrue(result)
        self.assertEqual(self.context_manager.get_context('key1'), 'value1')
        self.assertIsNone(self.context_manager.get_context('key2'))

    def test_restore_nonexistent_snapshot(self):
        """Test restoring non-existent snapshot."""
        result = self.context_manager.restore_context(999)
        self.assertFalse(result)

    # Test clear_context
    def test_clear_context(self):
        """Test clearing context."""
        self.context_manager.set_context('key1', 'value1')
        self.context_manager.set_context('key2', 'value2')
        self.context_manager.clear_context()
        self.assertEqual(self.context_manager.get_context(), {})

    # Test execute
    def test_execute_get_operation(self):
        """Test execute with get operation."""
        self.context_manager.set_context('test_key', 'test_value')
        result = self.context_manager.execute(operation='get', key='test_key')
        self.assertEqual(result, 'test_value')

    def test_execute_set_operation(self):
        """Test execute with set operation."""
        result = self.context_manager.execute(
            operation='set', 
            key='new_key', 
            value='new_value'
        )
        self.assertTrue(result)
        self.assertEqual(self.context_manager.get_context('new_key'), 'new_value')

    def test_execute_merge_operation(self):
        """Test execute with merge operation."""
        result = self.context_manager.execute(
            operation='merge',
            value={'key1': 'value1'}
        )
        self.assertEqual(result, {})
        self.assertEqual(self.context_manager.get_context('key1'), 'value1')

    def test_execute_clear_operation(self):
        """Test execute with clear operation."""
        self.context_manager.set_context('key1', 'value1')
        result = self.context_manager.execute(operation='clear')
        self.assertTrue(result)
        self.assertEqual(self.context_manager.get_context(), {})

    def test_execute_invalid_operation(self):
        """Test execute with invalid operation."""
        with self.assertRaises(ValueError):
            self.context_manager.execute(operation='invalid')

    def test_execute_missing_args(self):
        """Test execute with missing required arguments."""
        with self.assertRaises(ValueError):
            self.context_manager.execute(operation='set', key='key')
        with self.assertRaises(ValueError):
            self.context_manager.execute(operation='merge', value='not_a_dict')

    # Performance benchmarks
    def test_performance_get_context(self):
        """Benchmark get_context performance."""
        self.context_manager.set_context('perf_key', 'perf_value')
        time = timeit.timeit(
            lambda: self.context_manager.get_context('perf_key'),
            number=1000
        )
        self.assertLess(time, 0.1)  # Should take less than 100ms for 1000 ops

    def test_performance_set_context(self):
        """Benchmark set_context performance."""
        time = timeit.timeit(
            lambda: self.context_manager.set_context('perf_key', 'perf_value'),
            number=1000
        )
        self.assertLess(time, 0.1)

    def test_thread_safety(self):
        """Test thread safety of context operations."""
        import threading
        
        def worker():
            for i in range(100):
                self.context_manager.set_context(f'key_{i}', i)
                self.context_manager.get_context(f'key_{i}')
                if i % 10 == 0:
                    self.context_manager.snapshot_context()
        
        threads = [threading.Thread(target=worker) for _ in range(10)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        
        # Verify no data corruption occurred
        for i in range(100):
            self.assertEqual(self.context_manager.get_context(f'key_{i}'), i)

if __name__ == '__main__':
    unittest.main()