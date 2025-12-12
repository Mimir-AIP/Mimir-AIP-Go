import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ErrorBoundary, ErrorDisplay } from '../ErrorBoundary'

// Component that throws an error when rendered
function ThrowError({ shouldThrow, message }: { shouldThrow: boolean; message: string }) {
  if (shouldThrow) {
    throw new Error(message)
  }
  return <div>No error</div>
}

describe('ErrorBoundary', () => {
  // Suppress console.error for these tests since we're intentionally throwing errors
  const originalConsoleError = console.error
  beforeAll(() => {
    console.error = vi.fn()
  })
  
  afterAll(() => {
    console.error = originalConsoleError
  })

  describe('Error Catching', () => {
    it('should render children when there is no error', () => {
      render(
        <ErrorBoundary>
          <div>Test content</div>
        </ErrorBoundary>
      )
      
      expect(screen.getByText('Test content')).toBeInTheDocument()
    })

    it('should catch and display errors from children', () => {
      render(
        <ErrorBoundary>
          <ThrowError shouldThrow={true} message="Test error message" />
        </ErrorBoundary>
      )
      
      expect(screen.getByText('Something went wrong')).toBeInTheDocument()
      expect(screen.getByText('Test error message')).toBeInTheDocument()
    })

    it('should show default error message when error has no message', () => {
      const ErrorComponent = () => {
        throw new Error()
      }
      
      render(
        <ErrorBoundary>
          <ErrorComponent />
        </ErrorBoundary>
      )
      
      expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument()
    })
  })

  describe('Custom Fallback', () => {
    it('should render custom fallback when provided', () => {
      const customFallback = <div>Custom error UI</div>
      
      render(
        <ErrorBoundary fallback={customFallback}>
          <ThrowError shouldThrow={true} message="Error" />
        </ErrorBoundary>
      )
      
      expect(screen.getByText('Custom error UI')).toBeInTheDocument()
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
    })
  })

  describe('Error Recovery', () => {
    it('should have a retry button in default error UI', () => {
      render(
        <ErrorBoundary>
          <ThrowError shouldThrow={true} message="Error" />
        </ErrorBoundary>
      )
      
      const retryButton = screen.getByRole('button', { name: /try again/i })
      expect(retryButton).toBeInTheDocument()
    })

    it('should reset error state when try again is clicked', () => {
      let shouldThrow = true
      const TestComponent = () => {
        if (shouldThrow) {
          throw new Error('Error')
        }
        return <div>No error</div>
      }
      
      const { rerender } = render(
        <ErrorBoundary>
          <TestComponent />
        </ErrorBoundary>
      )
      
      // Error should be displayed
      expect(screen.getByText('Something went wrong')).toBeInTheDocument()
      
      // Click try again and update component to not throw
      const retryButton = screen.getByRole('button', { name: /try again/i })
      shouldThrow = false
      fireEvent.click(retryButton)
      
      // After clicking, error boundary resets, rerender with fixed component
      rerender(
        <ErrorBoundary>
          <TestComponent />
        </ErrorBoundary>
      )
      
      // Should show normal content now
      expect(screen.getByText('No error')).toBeInTheDocument()
      expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
    })
  })

  describe('Styling', () => {
    it('should have error styling in default UI', () => {
      const { container } = render(
        <ErrorBoundary>
          <ThrowError shouldThrow={true} message="Error" />
        </ErrorBoundary>
      )
      
      const card = container.querySelector('[data-slot="card"]')
      expect(card).toBeTruthy()
      const cardElement = card as HTMLElement
      expect(cardElement.className).toContain('border-red-500')
      expect(cardElement.className).toContain('bg-navy')
    })
  })
})

describe('ErrorDisplay', () => {
  describe('Basic Rendering', () => {
    it('should render error message', () => {
      render(<ErrorDisplay error="Test error message" />)
      
      expect(screen.getByText('Error')).toBeInTheDocument()
      expect(screen.getByText('Test error message')).toBeInTheDocument()
    })

    it('should not render retry button when onRetry is not provided', () => {
      render(<ErrorDisplay error="Test error" />)
      
      const retryButton = screen.queryByRole('button', { name: /retry/i })
      expect(retryButton).not.toBeInTheDocument()
    })

    it('should render retry button when onRetry is provided', () => {
      const mockRetry = vi.fn()
      render(<ErrorDisplay error="Test error" onRetry={mockRetry} />)
      
      const retryButton = screen.getByRole('button', { name: /retry/i })
      expect(retryButton).toBeInTheDocument()
    })
  })

  describe('Retry Functionality', () => {
    it('should call onRetry when retry button is clicked', () => {
      const mockRetry = vi.fn()
      render(<ErrorDisplay error="Test error" onRetry={mockRetry} />)
      
      const retryButton = screen.getByRole('button', { name: /retry/i })
      fireEvent.click(retryButton)
      
      expect(mockRetry).toHaveBeenCalledTimes(1)
    })

    it('should call onRetry multiple times when clicked multiple times', () => {
      const mockRetry = vi.fn()
      render(<ErrorDisplay error="Test error" onRetry={mockRetry} />)
      
      const retryButton = screen.getByRole('button', { name: /retry/i })
      fireEvent.click(retryButton)
      fireEvent.click(retryButton)
      fireEvent.click(retryButton)
      
      expect(mockRetry).toHaveBeenCalledTimes(3)
    })
  })

  describe('Styling', () => {
    it('should have error styling', () => {
      const { container } = render(<ErrorDisplay error="Test error" />)
      
      const card = container.querySelector('[data-slot="card"]')
      expect(card).toBeTruthy()
      const cardElement = card as HTMLElement
      expect(cardElement.className).toContain('border-red-500')
      expect(cardElement.className).toContain('bg-navy')
    })

    it('should style error heading with red text', () => {
      render(<ErrorDisplay error="Test error" />)
      
      const heading = screen.getByText('Error')
      expect(heading.className).toContain('text-red-500')
    })
  })

  describe('Edge Cases', () => {
    it('should handle empty error message', () => {
      render(<ErrorDisplay error="" />)
      
      expect(screen.getByText('Error')).toBeInTheDocument()
      // Empty error message should still render (empty string)
      const errorParagraph = screen.getByText((content, element) => {
        return element?.tagName.toLowerCase() === 'p' && content === ''
      })
      expect(errorParagraph).toBeInTheDocument()
    })

    it('should handle very long error messages', () => {
      const longError = 'A'.repeat(1000)
      render(<ErrorDisplay error={longError} />)
      
      expect(screen.getByText(longError)).toBeInTheDocument()
    })

    it('should handle error messages with special characters', () => {
      const specialError = '<script>alert("xss")</script>'
      render(<ErrorDisplay error={specialError} />)
      
      // React escapes special characters, so this should be safe
      expect(screen.getByText(specialError)).toBeInTheDocument()
    })
  })
})
