export default function ({
  type,
  value,
  placeholder,
  handleInputChange,
  handleKeyDown,
  disabled = false,
  error,
  hasDropdownSelection = true,
  optionType,
  options,
  handleSelectOption,
  selectedItem
}) {
  return (
    <div className={`border rounded-full px-[3px] py-[2px] ${error ? 'border-pink-500/40' : 'border-purple-50/20'}`}>
      <div className={`relative w-full px-4 py-3 border bg-transparent rounded-full focus:outline-none focus:ring focus:ring-cyan-600 disabled:opacity-30 ${error ? 'border-pink-500' : 'border-purple-50/40'}`}>
        <input
          autoFocus
          spellCheck='false'
          type={type}
          value={value}
          className={`block ${hasDropdownSelection ? 'w-10/12' : 'w-full'} sm:text-sm bg-transparent focus:outline-none`}
          placeholder={placeholder}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          disabled={disabled}
        />
        {hasDropdownSelection &&
          <div className='absolute inset-y-0 right-6 flex items-center'>
            <label htmlFor={optionType} className='sr-only'>
              {optionType}
            </label>
            <select
              id={optionType}
              name={optionType}
              onChange={handleSelectOption}
              value={selectedItem}
              className='h-full py-0 pl-2 border-transparent bg-transparent text-sm focus:outline-none'
            >
              {options.map((option) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          </div>}
      </div>
    </div>
  )
}
