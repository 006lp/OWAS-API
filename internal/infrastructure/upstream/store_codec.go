package upstream

import (
	"fmt"

	"owas-api/internal/domain"
	"owas-api/internal/protocol/protostuff"
)

const (
	classNavCard          = "com.heytap.appstore.watch.domain.v2.NavCardDto"
	classBannerCard       = "com.heytap.appstore.watch.domain.v2.BannerCardDto"
	classResourceListCard = "com.heytap.appstore.watch.domain.v2.ResourceListCardDto"
	classBaseCard         = "com.heytap.appstore.watch.domain.v2.CardDto"
)

func toDomainUnknown(f protostuff.RawField) domain.UnknownField {
	return domain.UnknownField{
		Tag:      f.Tag,
		WireType: f.WireType,
		Value:    f.PreviewValue(),
	}
}

func decodeMapInto(target *map[string]any, raw []byte) error {
	k, v, err := protostuff.DecodeMapEntry(raw)
	if err != nil {
		return err
	}
	if *target == nil {
		*target = make(map[string]any)
	}
	(*target)[k] = v
	return nil
}

// applyResourceField decodes ResourceDto / BaseDetailDto tags 1..33.
// Returns handled=false if the tag is not part of ResourceDto.
func applyResourceField(f protostuff.RawField, r *domain.Resource) (bool, error) {
	switch f.Tag {
	case 1:
		r.AppID = f.Int64()
	case 2:
		r.VerID = f.Int64()
	case 3:
		r.AppName = f.String()
	case 4:
		r.PkgName = f.String()
	case 5:
		r.VerName = f.String()
	case 6:
		r.VerCode = f.Int64()
	case 7:
		r.IconURL = f.String()
	case 8:
		r.Size = f.Int64()
	case 9:
		r.SizeDesc = f.String()
	case 10:
		r.DlCount = f.Int64()
	case 11:
		r.DlDesc = f.String()
	case 12:
		r.URL = f.String()
	case 13:
		r.Desc = f.String()
	case 14:
		r.Screenshots = append(r.Screenshots, f.String())
	case 15:
		r.MD5 = f.String()
	case 16:
		r.Checksum = f.String()
	case 17:
		r.ShortDesc = f.String()
	case 18:
		if f.WireType != protostuff.WireLen {
			return false, nil
		}
		if err := decodeMapInto(&r.Ext, f.Bytes); err != nil {
			return true, err
		}
	case 19:
		if f.WireType != protostuff.WireLen {
			return false, nil
		}
		if err := decodeMapInto(&r.Stat, f.Bytes); err != nil {
			return true, err
		}
	case 20:
		r.OnlineTime = f.Int64()
	case 21:
		r.Type = f.Int32()
	case 22:
		r.EnglishName = f.String()
	case 23:
		r.PrivacyPolicy = f.String()
	case 24:
		r.PrivacyRights = f.String()
	case 25:
		r.VersionExplain = f.String()
	case 26:
		r.IPhoneLink = f.Int32()
	case 27:
		r.IPhoneLinkDetail = f.String()
	case 28:
		r.QuickAppURL = f.String()
	case 29:
		r.AgeRating = f.String()
	case 30:
		r.Permission = f.String()
	case 31:
		r.Payment = f.Int32()
	case 32:
		r.Price = f.Int32()
	case 33:
		r.IsBought = f.Int32()
	default:
		return false, nil
	}
	return true, nil
}

func decodeResource(data []byte) (domain.Resource, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return domain.Resource{}, err
	}
	var res domain.Resource
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		handled, err := applyResourceField(f, &res)
		if err != nil {
			return domain.Resource{}, err
		}
		if !handled {
			res.UnknownFields = append(res.UnknownFields, toDomainUnknown(f))
		}
	}
	return res, nil
}

func decodeSearchResult(data []byte) (*domain.SearchResult, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.SearchResult{
		Apps: make([]domain.Resource, 0),
	}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			app, err := decodeResource(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode SearchResult.apps: %w", err)
			}
			out.Apps = append(out.Apps, app)
		case 2:
			out.AllMatchApp = f.String()
		case 3:
			out.IsEnd = f.Int32()
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func decodeDeveloper(data []byte) (*domain.Developer, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	dev := &domain.Developer{}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			dev.ReleaseTime = f.Int64()
		case 2:
			dev.PkgPermiss = f.String()
		case 3:
			dev.Developer = f.String()
		default:
			dev.UnknownFields = append(dev.UnknownFields, toDomainUnknown(f))
		}
	}
	return dev, nil
}

func decodeCategoryPermission(data []byte) (*domain.CategoryPermission, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	perm := &domain.CategoryPermission{
		BasePermission:    make([]string, 0),
		PrivacyPermission: make([]string, 0),
	}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			perm.BasePermission = append(perm.BasePermission, f.String())
		case 2:
			perm.PrivacyPermission = append(perm.PrivacyPermission, f.String())
		default:
			perm.UnknownFields = append(perm.UnknownFields, toDomainUnknown(f))
		}
	}
	return perm, nil
}

func decodeAppDetail(data []byte) (*domain.AppDetail, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.AppDetail{}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			base, err := decodeResource(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode AppDetail.base: %w", err)
			}
			out.Base = &base
		case 2:
			dev, err := decodeDeveloper(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode AppDetail.developer: %w", err)
			}
			out.Developer = dev
		case 3:
			perm, err := decodeCategoryPermission(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode AppDetail.permissionDto: %w", err)
			}
			out.PermissionDto = perm
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func decodeAppDownload(data []byte) (*domain.AppDownload, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.AppDownload{}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			out.AppID = f.Int64()
		case 2:
			out.VerID = f.Int64()
		case 3:
			out.DownURL = f.String()
		case 4:
			out.MD5 = f.String()
		case 5:
			out.HeaderMD5 = f.String()
		case 6:
			out.PkgName = f.String()
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func decodeBanner(data []byte) (domain.Banner, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return domain.Banner{}, err
	}
	var b domain.Banner
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			b.ID = f.Int64()
		case 2:
			b.Image = f.String()
		case 3:
			b.ImageWidth = f.Int32()
		case 4:
			b.ImageHeight = f.Int32()
		case 5:
			b.ActionParam = f.String()
		case 6:
			b.Title = f.String()
		case 7:
			b.Desc = f.String()
		case 8:
			b.StartTime = f.Int64()
		case 9:
			b.EndTime = f.Int64()
		case 10:
			b.Sort = f.Int32()
		default:
			b.UnknownFields = append(b.UnknownFields, toDomainUnknown(f))
		}
	}
	return b, nil
}

func applyCardBaseField(f protostuff.RawField, c *domain.Card) (bool, error) {
	switch f.Tag {
	case 1:
		c.Code = f.Int32()
	case 2:
		c.Key = f.Int64()
	case 3:
		c.ActionParam = f.String()
	case 4:
		c.CornerLabel = f.Int32()
	case 5:
		if f.WireType != protostuff.WireLen {
			return false, nil
		}
		if err := decodeMapInto(&c.Ext, f.Bytes); err != nil {
			return true, err
		}
	case 6:
		if f.WireType != protostuff.WireLen {
			return false, nil
		}
		if err := decodeMapInto(&c.Stat, f.Bytes); err != nil {
			return true, err
		}
	default:
		return false, nil
	}
	return true, nil
}

func decodeCard(data []byte) (domain.Card, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return domain.Card{}, err
	}
	className := protostuff.ExtractPolyClassName(fields)
	card := domain.Card{Type: className}

	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		handled, err := applyCardBaseField(f, &card)
		if err != nil {
			return domain.Card{}, err
		}
		if handled {
			continue
		}

		switch className {
		case classNavCard:
			switch f.Tag {
			case 101:
				banner, err := decodeBanner(f.Bytes)
				if err != nil {
					return domain.Card{}, fmt.Errorf("decode NavCard.banners: %w", err)
				}
				card.Banners = append(card.Banners, banner)
			case 102:
				item, err := decodeBanner(f.Bytes)
				if err != nil {
					return domain.Card{}, fmt.Errorf("decode NavCard.items: %w", err)
				}
				card.Items = append(card.Items, item)
			default:
				card.UnknownFields = append(card.UnknownFields, toDomainUnknown(f))
			}

		case classBannerCard:
			switch f.Tag {
			case 101:
				card.Title = f.String()
			case 102:
				card.Desc = f.String()
			case 103:
				banner, err := decodeBanner(f.Bytes)
				if err != nil {
					return domain.Card{}, fmt.Errorf("decode BannerCard.banners: %w", err)
				}
				card.Banners = append(card.Banners, banner)
			case 104:
				app, err := decodeResource(f.Bytes)
				if err != nil {
					return domain.Card{}, fmt.Errorf("decode BannerCard.apps: %w", err)
				}
				card.Apps = append(card.Apps, app)
			case 105:
				card.BgImageURL = f.String()
			default:
				card.UnknownFields = append(card.UnknownFields, toDomainUnknown(f))
			}

		case classResourceListCard:
			switch f.Tag {
			case 101:
				card.Title = f.String()
			case 102:
				card.Desc = f.String()
			case 103:
				res, err := decodeResource(f.Bytes)
				if err != nil {
					return domain.Card{}, fmt.Errorf("decode ResourceListCard.resources: %w", err)
				}
				card.Resources = append(card.Resources, res)
			default:
				card.UnknownFields = append(card.UnknownFields, toDomainUnknown(f))
			}

		case classBaseCard, "":
			card.UnknownFields = append(card.UnknownFields, toDomainUnknown(f))

		default:
			card.UnknownFields = append(card.UnknownFields, toDomainUnknown(f))
		}
	}

	return card, nil
}

func decodeViewLayer(data []byte) (*domain.ViewLayer, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.ViewLayer{
		Cards: make([]domain.Card, 0),
	}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			out.Title = f.String()
		case 2:
			out.PageKey = f.Int32()
		case 3:
			card, err := decodeCard(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode ViewLayer.cards: %w", err)
			}
			out.Cards = append(out.Cards, card)
		case 4:
			out.IsEnd = f.Int32()
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func encodeUpgradeWrapReq(apps []domain.InstalledApp) []byte {
	enc := protostuff.NewEncoder()
	for _, app := range apps {
		item := protostuff.NewEncoder().
			AppendString(1, app.PkgName).
			AppendString(2, app.VerCode).
			Bytes()
		enc.AppendMessage(1, item)
	}
	return enc.Bytes()
}

func decodeUpgradeItem(data []byte) (domain.UpgradeItem, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return domain.UpgradeItem{}, err
	}
	var item domain.UpgradeItem
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		handled, err := applyResourceField(f, &item.Resource)
		if err != nil {
			return domain.UpgradeItem{}, err
		}
		if handled {
			continue
		}
		switch f.Tag {
		case 101:
			item.UpdateDesc = f.String()
		case 102:
			item.UpgradeFlag = f.Int32()
		default:
			item.UnknownFields = append(item.UnknownFields, toDomainUnknown(f))
		}
	}
	return item, nil
}

func decodeUpgradeResult(data []byte) (*domain.UpgradeResult, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.UpgradeResult{
		Upgrades: make([]domain.UpgradeItem, 0),
	}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			item, err := decodeUpgradeItem(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode UpgradeResult.upgrades: %w", err)
			}
			out.Upgrades = append(out.Upgrades, item)
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func decodeProtocol(data []byte) (*domain.Protocol, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.Protocol{}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			out.Privacy = f.String()
		case 2:
			out.Agreement = f.String()
		case 3:
			out.PrivacyVersion = f.Int32()
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func encodeOrderPayParam(in domain.CreateOrderInput) []byte {
	enc := protostuff.NewEncoder().
		AppendInt64(1, in.AppID).
		AppendInt32(2, in.Price).
		AppendString(3, in.ProductName).
		AppendString(4, in.ProductDesc).
		AppendString(5, in.Token)
	if in.Model != "" {
		enc.AppendString(6, in.Model)
	}
	if in.IP != "" {
		enc.AppendString(7, in.IP)
	}
	return enc.Bytes()
}

func decodeOrderPay(data []byte) (*domain.OrderPay, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.OrderPay{}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			out.OrderID = f.String()
		case 2:
			out.PayURL = f.String()
		case 3:
			out.ExpireTime = f.String()
		case 4:
			out.PrePayToken = f.String()
		case 5:
			out.PartnerCode = f.String()
		case 6:
			out.UserID = f.String()
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func encodeOrderParam(in domain.QueryOrderInput) []byte {
	enc := protostuff.NewEncoder().
		AppendString(1, in.OrderID).
		AppendString(2, in.UserID).
		AppendInt64(3, in.AppID)
	if in.Token != "" {
		enc.AppendString(4, in.Token)
	}
	if in.Model != "" {
		enc.AppendString(5, in.Model)
	}
	if in.IP != "" {
		enc.AppendString(6, in.IP)
	}
	return enc.Bytes()
}

func decodeOrderStatus(data []byte) (*domain.OrderStatus, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.OrderStatus{}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			out.AppID = f.Int64()
		case 2:
			out.PayStatus = f.Int32()
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func decodeOrderStatusList(data []byte) (*domain.OrderStatusList, error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.OrderStatusList{
		StatusVOList: make([]domain.OrderStatus, 0),
	}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			st, err := decodeOrderStatus(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode OrderStatusList.statusVOList: %w", err)
			}
			out.StatusVOList = append(out.StatusVOList, *st)
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}

func decodePayResponse[T any](data []byte, decodeData func([]byte) (*T, error)) (*domain.PayResponse[T], error) {
	fields, err := protostuff.ParseRawFields(data)
	if err != nil {
		return nil, err
	}
	out := &domain.PayResponse[T]{}
	for _, f := range fields {
		if f.Tag == protostuff.PolyClassTag && f.WireType == protostuff.WireLen {
			continue
		}
		switch f.Tag {
		case 1:
			out.Code = f.String()
		case 2:
			out.Msg = f.String()
		case 3:
			item, err := decodeData(f.Bytes)
			if err != nil {
				return nil, fmt.Errorf("decode PayResponse.data: %w", err)
			}
			out.Data = item
		default:
			out.UnknownFields = append(out.UnknownFields, toDomainUnknown(f))
		}
	}
	return out, nil
}
