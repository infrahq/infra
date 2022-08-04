import { Combobox } from '@headlessui/react'
import { useRef } from 'react'

import Badge from './badge'
import ComboboxItem from './combobox-item'

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
            className='relative my-2 w-full bg-transparent text-xs text-gray-300 placeholder:italic focus:outline-none'
            onChange={e => setQuery(e.target.value)}
            onFocus={() => {
              button.current?.click()
            }}
            onKeyDown={e => onKeyDownEvent(e.key)}
            placeholder={selectedEmails.length === 0 ? 'Add email here' : ''}
          />
        </div>
      </div>
      {filteredEmail.length > 0 && (
        <Combobox.Options className='absolute -left-[13px] z-10 mt-1 max-h-60 w-56 overflow-auto rounded-md border border-gray-700 bg-gray-800 py-1 text-2xs ring-1 ring-black ring-opacity-5 focus:outline-none'>
          {filteredEmail?.map(f => (
            <Combobox.Option
              key={f.id}
              value={f}
              className={({ active }) =>
                `relative cursor-default select-none py-2 px-3 hover:bg-gray-700 ${
                  active ? 'bg-gray-700' : ''
                }`
              }
            >
              <ComboboxItem
                title={f.name}
                subtitle={f.user ? 'User' : f.group ? 'Group' : ''}
              />
            </Combobox.Option>
          ))}
        </Combobox.Options>
      )}
      <Combobox.Button className='hidden' ref={button} />
    </Combobox>
  )
}
