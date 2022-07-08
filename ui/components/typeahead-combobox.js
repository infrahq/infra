import { Combobox } from '@headlessui/react'
import { useRef } from 'react'

import Badge from './badge'
import TypeaheadDropdown from './typeahead-dropdown'

export default function TypeaheadCombobox({
  selectedEmails,
  setSelectedEmails,
  onRemove,
  inputRef,
  setQuery,
  filteredEmail,
  onKeyDownEvent,
}) {
  const button = useRef()

  return (
    <Combobox
      as='div'
      className='relative flex-1'
      onChange={e => {
        setSelectedEmails([...selectedEmails, e])
      }}
    >
      <div className='flex flex-auto flex-wrap'>
        {selectedEmails?.map(i => (
          <Badge key={i.id} onRemove={() => onRemove(i)}>
            {i.name}
          </Badge>
        ))}
        <div className='flex-1'>
          <Combobox.Input
            type='search'
            ref={inputRef}
            className='relative w-full bg-transparent text-xs text-gray-300 placeholder:italic focus:outline-none'
            onChange={e => setQuery(e.target.value)}
            onFocus={() => {
              button.current?.click()
            }}
            onKeyDown={e => onKeyDownEvent(e.key)}
            placeholder={selectedEmails.length === 0 ? 'Add email here' : ''}
          />
        </div>
      </div>
      <TypeaheadDropdown filtered={filteredEmail} />
      <Combobox.Button className='hidden' ref={button} />
    </Combobox>
  )
}
