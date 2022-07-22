import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'

import Notification from '../../components/notification'

describe('Notification Component', () => {
  it('should render', () => {
    expect(() =>
      render(<Notification show={true} setShow={() => {}} text='test text' />)
    ).not.toThrow()
  })

  it('should show with correct text', () => {
    const text = 'test text'
    const { queryByText } = render(
      <Notification show={true} setShow={() => {}} text={text} />
    )

    expect(queryByText(text)).toBeInTheDocument()
  })

  it('should fire button onClick to setShow be false', () => {
    const setShow = jest.fn(() => {})
    render(<Notification show={true} setShow={setShow} text='test text' />)

    fireEvent.click(screen.getByTestId('notification-remove-button'))
    expect(setShow).toHaveBeenCalledTimes(1)
    expect(setShow).toHaveBeenCalledWith(false)
  })
})
