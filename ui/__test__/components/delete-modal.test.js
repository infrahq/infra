import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'

import DeleteModal from '../../components/delete-modal'

const DeleteModalProps = {
  open: true,
  setOpen: jest.fn(() => {}),
  onSubmit: jest.fn(() => {}),
  title: 'delete modal title',
  message: 'delete modal message',
}

jest.mock('@headlessui/react', () => ({
  ...jest.requireActual('@headlessui/react'),
}))

global.IntersectionObserver = jest.fn(() => ({
  observe: () => {},
  unobserve: () => {},
  disconnect: () => {},
}))

describe('Delete Modal Component', () => {
  it('should render', () => {
    expect(() =>
      render(
        <DeleteModal
          open={DeleteModalProps.open}
          setOpen={DeleteModalProps.setOpen}
          onSubmit={DeleteModalProps.onSubmit}
          title={DeleteModalProps.title}
          message={DeleteModalProps.message}
        />
      )
    ).not.toThrow()
  })

  it('should not show the modal', () => {
    const { queryByTestId } = render(
      <DeleteModal
        open={false}
        setOpen={DeleteModalProps.setOpen}
        onSubmit={DeleteModalProps.onSubmit}
        title={DeleteModalProps.title}
        message={DeleteModalProps.message}
      />
    )

    expect(queryByTestId('delete-modal')).not.toBeInTheDocument()
  })

  it('should show the modal with correct title and message', () => {
    const { queryByTestId } = render(
      <DeleteModal
        open={DeleteModalProps.open}
        setOpen={DeleteModalProps.setOpen}
        onSubmit={DeleteModalProps.onSubmit}
        title={DeleteModalProps.title}
        message={DeleteModalProps.message}
      />
    )

    expect(queryByTestId('delete-modal')).toBeInTheDocument()
    expect(screen.getByText(DeleteModalProps.title)).toBeInTheDocument()
    expect(screen.getByText(DeleteModalProps.message)).toBeInTheDocument()
  })

  it('should fire correct button onClick', () => {
    render(
      <DeleteModal
        open={DeleteModalProps.open}
        setOpen={DeleteModalProps.setOpen}
        onSubmit={DeleteModalProps.onSubmit}
        title={DeleteModalProps.title}
        message={DeleteModalProps.message}
      />
    )

    fireEvent.click(screen.getByTestId('delete-modal-primary-button'))
    expect(DeleteModalProps.onSubmit).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByTestId('delete-modal-cancel-button'))
    expect(DeleteModalProps.setOpen).toHaveBeenCalledWith(false)
  })
})
