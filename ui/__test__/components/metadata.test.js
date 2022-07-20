import React from 'react'
import { render } from '@testing-library/react'

import Metadata from '../../components/metadata'

const mockedData = [
  { title: 'test title 1', data: 'test data 1' },
  { title: 'test title 2', data: 'test data 2' },
  { title: 'test title 3', data: 'test data 3' },
]

describe('Metadata Component', () => {
  it('should render', () => {
    expect(() => render(<Metadata data={mockedData} />)).not.toThrow()
  })

  it('should render correct data', () => {
    const { getAllByTestId } = render(<Metadata data={mockedData} />)

    const items = getAllByTestId('metadata-item')
    const title = getAllByTestId('metadata-title')
    const data = getAllByTestId('metadata-data')

    expect(items.length).toBe(mockedData.length)
    expect(title[0]).toHaveTextContent(mockedData[0].title)
    expect(data[0]).toHaveTextContent(mockedData[0].data)
  })
})
