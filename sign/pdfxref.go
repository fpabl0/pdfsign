package sign

import (
	"errors"
	"strconv"
)

func (context *SignContext) writeXref() error {

	// @todo: support stream xref.

	if context.PDFReader.XrefInformation.Type == "table" {
		if err := context.writeXrefTable(); err != nil {
			return err
		}
	} else {
		return errors.New("Unkwn xref type: " + context.PDFReader.XrefInformation.Type)
	}

	return nil
}

func (context *SignContext) writeXrefTable() error {
	xref_size := "xref\n0 " + strconv.FormatInt(context.PDFReader.XrefInformation.ItemCount, 10)
	new_xref_size := "xref\n0 " + strconv.FormatInt(context.PDFReader.XrefInformation.ItemCount+2, 10)

	if _, err := context.OutputFile.Write([]byte(new_xref_size)); err != nil {
		return err
	}

	// Write the old xref table to the output pdf.
	if err := writePartFromSourceFileToTargetFile(context.InputFile, context.OutputFile, context.PDFReader.XrefInformation.StartPos+int64(len(xref_size)), context.PDFReader.XrefInformation.Length-int64(len(xref_size))); err != nil {
		return err
	}

	// Create the new catalog xref line.
	catalog_object_start_position := strconv.FormatInt(context.PDFReader.XrefInformation.StartPos, 10)
	catalog_xref_line := leftPad(catalog_object_start_position, "0", 10-len(catalog_object_start_position)) + " 00000 n\n"

	// Write the new catalog xref line.
	if _, err := context.OutputFile.Write([]byte(catalog_xref_line)); err != nil {
		return err
	}

	// Create the new signature xref line.
	signature_object_start_position := strconv.FormatInt(context.PDFReader.XrefInformation.StartPos+context.CatalogData.Length, 10)
	signature_xref_line := leftPad(signature_object_start_position, "0", 10-len(signature_object_start_position)) + " 00000 n\n"

	// Write the new signature xref line.
	if _, err := context.OutputFile.Write([]byte(signature_xref_line)); err != nil {
		return err
	}

	return nil
}
