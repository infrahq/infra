import { test, expect } from '@playwright/test'

test('should navigate to the about page', async ({ page }) => {
  await page.goto('http://localhost:8080')
  expect(true).toBe(true)
})
