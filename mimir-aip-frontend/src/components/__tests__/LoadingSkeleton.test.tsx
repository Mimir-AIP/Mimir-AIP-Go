import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { CardListSkeleton, TableSkeleton, DetailsSkeleton } from '../LoadingSkeleton'

describe('LoadingSkeleton Components', () => {
  describe('CardListSkeleton', () => {
    it('should render default number of skeleton cards', () => {
      const { container } = render(<CardListSkeleton />)
      const cards = container.querySelectorAll('[data-slot="card"]')
      // Default count is 3
      expect(cards.length).toBe(3)
    })

    it('should render custom number of skeleton cards', () => {
      const { container } = render(<CardListSkeleton count={5} />)
      const cards = container.querySelectorAll('[data-slot="card"]')
      expect(cards.length).toBe(5)
    })

    it('should render skeleton elements within each card', () => {
      const { container } = render(<CardListSkeleton count={1} />)
      // Each card should have multiple skeleton elements (title, content, buttons)
      const skeletons = container.querySelectorAll('[class*="animate-pulse"]')
      expect(skeletons.length).toBeGreaterThan(0)
    })

    it('should use grid layout classes', () => {
      const { container } = render(<CardListSkeleton />)
      const gridContainer = container.firstChild
      expect(gridContainer?.nodeName).toBe('DIV')
      const gridElement = gridContainer as HTMLElement
      expect(gridElement.className).toContain('grid')
    })
  })

  describe('TableSkeleton', () => {
    it('should render default number of skeleton rows', () => {
      const { container } = render(<TableSkeleton />)
      const rows = container.querySelectorAll('[class*="animate-pulse"]')
      // Default rows is 5
      expect(rows.length).toBe(5)
    })

    it('should render custom number of skeleton rows', () => {
      const { container } = render(<TableSkeleton rows={10} />)
      const rows = container.querySelectorAll('[class*="animate-pulse"]')
      expect(rows.length).toBe(10)
    })

    it('should render single row when count is 1', () => {
      const { container } = render(<TableSkeleton rows={1} />)
      const rows = container.querySelectorAll('[class*="animate-pulse"]')
      expect(rows.length).toBe(1)
    })

    it('should use space-y layout for rows', () => {
      const { container } = render(<TableSkeleton />)
      const rowsContainer = container.firstChild
      expect(rowsContainer?.nodeName).toBe('DIV')
      const rowsElement = rowsContainer as HTMLElement
      expect(rowsElement.className).toContain('space-y')
    })
  })

  describe('DetailsSkeleton', () => {
    it('should render details skeleton structure', () => {
      const { container } = render(<DetailsSkeleton />)
      const skeletons = container.querySelectorAll('[class*="animate-pulse"]')
      // Should have multiple skeleton elements (title, labels, values, buttons)
      expect(skeletons.length).toBeGreaterThan(5)
    })

    it('should render within a Card component', () => {
      const { container } = render(<DetailsSkeleton />)
      const card = container.querySelector('[data-slot="card"]')
      expect(card).toBeTruthy()
    })

    it('should have skeleton elements for title, fields, and actions', () => {
      const { container } = render(<DetailsSkeleton />)
      const skeletons = container.querySelectorAll('[class*="animate-pulse"]')
      
      // Check that we have different sized skeletons (title, labels, content)
      const hasDifferentSizes = Array.from(skeletons).some((skeleton) => {
        const classes = (skeleton as HTMLElement).className
        return classes.includes('h-8') || classes.includes('h-4') || classes.includes('h-6')
      })
      
      expect(hasDifferentSizes).toBe(true)
    })

    it('should render action button skeletons', () => {
      const { container } = render(<DetailsSkeleton />)
      const skeletons = container.querySelectorAll('[class*="animate-pulse"]')
      
      // Button skeletons should be h-10
      const hasButtonSkeletons = Array.from(skeletons).some((skeleton) => {
        const classes = (skeleton as HTMLElement).className
        return classes.includes('h-10')
      })
      
      expect(hasButtonSkeletons).toBe(true)
    })
  })

  describe('Skeleton Styling', () => {
    it('CardListSkeleton should have navy background theme', () => {
      const { container } = render(<CardListSkeleton count={1} />)
      const card = container.querySelector('[data-slot="card"]')
      expect(card).toBeTruthy()
      const cardElement = card as HTMLElement
      expect(cardElement.className).toContain('bg-navy')
    })

    it('DetailsSkeleton should have navy background theme', () => {
      const { container } = render(<DetailsSkeleton />)
      const card = container.querySelector('[data-slot="card"]')
      expect(card).toBeTruthy()
      const cardElement = card as HTMLElement
      expect(cardElement.className).toContain('bg-navy')
    })

    it('Skeleton elements should have blue/20 opacity background', () => {
      const { container } = render(<CardListSkeleton count={1} />)
      const skeleton = container.querySelector('[class*="animate-pulse"]')
      expect(skeleton).toBeTruthy()
      const skeletonElement = skeleton as HTMLElement
      expect(skeletonElement.className).toContain('bg-blue/20')
    })
  })

  describe('Edge Cases', () => {
    it('CardListSkeleton should handle zero count', () => {
      const { container } = render(<CardListSkeleton count={0} />)
      const cards = container.querySelectorAll('[data-slot="card"]')
      expect(cards.length).toBe(0)
    })

    it('TableSkeleton should handle zero rows', () => {
      const { container } = render(<TableSkeleton rows={0} />)
      const rows = container.querySelectorAll('[class*="animate-pulse"]')
      expect(rows.length).toBe(0)
    })

    it('CardListSkeleton should handle large count', () => {
      const { container } = render(<CardListSkeleton count={100} />)
      const cards = container.querySelectorAll('[data-slot="card"]')
      expect(cards.length).toBe(100)
    })
  })
})
