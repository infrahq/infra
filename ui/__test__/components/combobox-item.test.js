import React from 'react'
import { render, screen } from '@testing-library/react'
import ComboboxItem from '../../components/combobox-item'

const title = 'test title'
const subtitle = 'test subtitle'

describe('Combobx Item Component', () => {
  it('should render', () => {
    expect(() => render(<ComboboxItem title={title} subtitle={subtitle} />))
  })

  it('should render with correct title and subtitle', () => {
    render(<ComboboxItem title={title} subtitle={subtitle} />)

    expect(screen.getByText(title)).toBeInTheDocument()
    expect(screen.getByText(subtitle)).toBeInTheDocument()
    expect(screen.queryByTestId('selectedIcon')).not.toBeInTheDocument()
  })

  it('should render check icon', () => {
    render(<ComboboxItem title={title} subtitle={subtitle} selected />)

    expect(screen.queryByTestId('selectedIcon')).toBeInTheDocument()
  })
})
